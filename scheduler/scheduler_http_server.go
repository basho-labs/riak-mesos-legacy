package framework

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
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
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
	} else {
		json.NewEncoder(w).Encode(schttp.sc.mgr.CreateNode(cluster))
	}
}

func (schttp *SchedulerHTTPServer) serveClusters(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(schttp.sc.clusters)
}

func (schttp *SchedulerHTTPServer) createCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	_, assigned := schttp.sc.clusters[clusterName]
	log.Info("CREATE CLUSTER: ")
	if assigned {
		w.WriteHeader(409)
	} else {
		cluster := schttp.sc.mgr.CreateCluster(clusterName)
		json.NewEncoder(w).Encode(cluster)
	}
}

func (schttp *SchedulerHTTPServer) getCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.clusters[clusterName]

	if !assigned {
		http.NotFound(w, r)
	} else {
		json.NewEncoder(w).Encode(cluster)
	}

}

func (schttp *SchedulerHTTPServer) serveNodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodes := schttp.sc.clusters[clusterName].nodes
	json.NewEncoder(w).Encode(nodes)
}

type customServer struct {}
func (customServer) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	log.Infof("%v %s %s %s ? %s %s %s", request.Host, request.RemoteAddr, request.Method, request.URL.Path, request.URL.RawQuery, request.Proto, request.Header.Get("User-Agent"))
	data, err := Asset("data/" + request.URL.Path)
	if err != nil {
		log.Panic(err)
	}
	w.Write(data)
}
func ServeExecutorArtifact(sc *SchedulerCore, schedulerHostname string) *SchedulerHTTPServer {
	ln, err := net.Listen("tcp", ":0")
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


	// This rewrites /static/FOO -> FOO
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", customServer{}))
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
	router.HandleFunc("/clusters", schttp.serveClusters)
	router.Methods("POST", "PUT").Path("/clusters/{cluster}").HandlerFunc(schttp.createCluster)
	router.Methods("GET").Path("/clusters/{cluster}").HandlerFunc(schttp.getCluster)
	router.Methods("GET").Path("/clusters/{cluster}/nodes").HandlerFunc(schttp.serveNodes)
	router.Methods("POST").Path("/clusters/{cluster}/nodes").HandlerFunc(schttp.createNode)

	//http.Serve(ln, newHandler())
	go http.Serve(ln, router)

	return schttp
}
