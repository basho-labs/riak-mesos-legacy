package framework

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"net/http/pprof"
	"os"
)

type SchedulerHTTPServer struct {
	hostURI      string
	riakURI      string
	executorName string
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

func ServeExecutorArtifact(schedulerHostname string) *SchedulerHTTPServer {
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

	executorName := "executor"

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
	hostURI := fmt.Sprintf("http://%s:%d/%s", hostname, port, executorName)
	riakURI := fmt.Sprintf("http://%s:%d/static/riak.tar.gz", hostname, port)

	fs := http.FileServer(http.Dir("."))
	//Info.Printf("Hosting artifact '%s' at '%s'", path, hostURI)
	log.Println("Serving at HostURI: ", hostURI)

	mux := http.NewServeMux()
	mux.HandleFunc("/executor", func(w http.ResponseWriter, request *http.Request) {
		log.Printf("%v %s %s %s ? %s %s %s", request.Host, request.RemoteAddr, request.Method, request.URL.Path, request.URL.RawQuery, request.Proto, request.Header.Get("User-Agent"))
		http.ServeFile(w, request, "./executor_linux_amd64")
	})
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	//http.Serve(ln, newHandler())
	go http.Serve(ln, mux)

	return &SchedulerHTTPServer{hostURI: hostURI, riakURI: riakURI, executorName: "./" + executorName}
}
