package scheduler

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"net/http/httputil"
	"net/url"
)

type SchedulerHTTPServer struct {
	sc           *SchedulerCore
	hostURI      string
	riakURI      string
	executorName string
	URI          string
}

func parseIP(address string) net.IP {
	addr, err := net.LookupIP(address)
	if err != nil {
		log.Fatal(err)
	}
	if len(addr) < 1 {
		log.Fatalf("failed to parse IP from address '%v'", address)
	}
	return addr[0]
}

func (schttp *SchedulerHTTPServer) createNode(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	frn := NewFrameworkRiakNode(schttp.sc.frameworkName)
	schttp.sc.schedulerState.Nodes[frn.UUID.String()] = frn
	schttp.sc.schedulerState.Persist()
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(frn)

}

func (schttp *SchedulerHTTPServer) serveNodes(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	json.NewEncoder(w).Encode(schttp.sc.schedulerState.Nodes)
}
func (schttp *SchedulerHTTPServer) GetURI() string {
	return schttp.URI
}

func (schttp *SchedulerHTTPServer) healthcheck(w http.ResponseWriter, r *http.Request) {
	var pass bool = true
	// TODO: Add better healthchecking
	rexc := schttp.sc.rex.NewRiakExplorerClient()
	_, err := rexc.Ping()
	if err == nil {
		fmt.Fprintln(w, "REX Client: OK")
	} else {
		fmt.Fprintln(w, "REX Client: NOT OK")
	}
	if pass {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
}

func ServeExecutorArtifact(sc *SchedulerCore, schedulerHostname string) *SchedulerHTTPServer {
	// When starting scheduler from Marathon, PORT0-N env vars will be set
	strBindPort := os.Getenv("PORT0")

	// If PORT0 isn't set, automatically bind to an available one
	// TODO: Sargun fix me
	if strBindPort == "" {
		strBindPort = "0"
	}
	ln, err := net.Listen("tcp", ":"+strBindPort)
	if err != nil {
		log.Fatal(err)
	}
	_, strPort, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		log.Fatal(err)
	}

	var hostname string

	if schedulerHostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		hostname = schedulerHostname
	}

	//TODO: MAKE THIS SMARTER
	//We need to ideally embed the executor into the scheduler
	//And we need to intelligently choose / decompress the executor based upon the host OS
	//This is a HACK.
	hostURI := fmt.Sprintf("http://%s:%d/static/executor_linux_amd64", hostname, port)
	riakURI := fmt.Sprintf("http://%s:%d/static/riak_linux_amd64.tar.gz", hostname, port)
	URI := fmt.Sprintf("http://%s:%d/", hostname, port)
	//Info.Printf("Hosting artifact '%s' at '%s'", path, hostURI)
	log.Println("Serving at HostURI: ", hostURI)

	router := mux.NewRouter().StrictSlash(true)

	fs := http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""})

	// This rewrites /static/FOO -> FOO
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	debugMux := http.NewServeMux()
	router.PathPrefix("/debug").Handler(debugMux)
	debugMux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	debugMux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	debugMux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	debugMux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))

	schttp := &SchedulerHTTPServer{
		sc:           sc,
		hostURI:      hostURI,
		riakURI:      riakURI,
		executorName: "./executor_linux_amd64",
		URI:          URI,
	}
	router.Methods("GET").Path("/api/v1/nodes").HandlerFunc(schttp.serveNodes)
	router.Methods("POST").Path("/api/v1/nodes").HandlerFunc(schttp.createNode)
	router.Methods("GET").Path("/healthcheck").HandlerFunc(schttp.healthcheck)

	rexURL := &url.URL{
		Host:   fmt.Sprintf("localhost:%d", sc.rexPort),
		Scheme: "http",
		Path:   "/",
	}
	router.PathPrefix("/").Handler(httputil.NewSingleHostReverseProxy(rexURL))

	//http.Serve(ln, newHandler())
	middleWare := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		log.Infof("%v %s %s %s ? %s %s %s", request.Host, request.RemoteAddr, request.Method, request.URL.Path, request.URL.RawQuery, request.Proto, request.Header.Get("User-Agent"))
		router.ServeHTTP(w, request)
	})

	log.Println("Listener Info: ", ln.Addr().String())

	// go http.ListenAndServe(":8080", nil)
	go http.Serve(ln, middleWare)

	return schttp
}
