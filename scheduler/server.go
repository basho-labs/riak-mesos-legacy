package scheduler

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/artifacts"
	rexclient "github.com/basho-labs/riak-mesos/riak_explorer"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"io/ioutil"
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

func (schttp *SchedulerHTTPServer) createCluster(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	_, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	log.Info("CREATE CLUSTER: ", clusterName)
	if assigned {
		w.WriteHeader(409)
	} else {
		cluster := NewFrameworkRiakCluster(clusterName)
		schttp.sc.schedulerState.Clusters[clusterName] = cluster
		schttp.sc.schedulerState.Persist()
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(cluster)
	}
}

func (schttp *SchedulerHTTPServer) restartCluster(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	log.Info("RESTART CLUSTER: ", clusterName)
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
	} else {
		cluster.RollingRestart()
		schttp.sc.schedulerState.Persist()
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(cluster)
	}
}

func (schttp *SchedulerHTTPServer) removeCluster(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
	} else {
		for _, node := range cluster.Nodes {
			node.KillNext()
		}
		cluster.KillNext()
		schttp.sc.schedulerState.Persist()
		w.WriteHeader(202)
	}
}

func (schttp *SchedulerHTTPServer) setConfig(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read file: ", err)
		return
	}
	cluster.RiakConfig = string(data)
	if err := schttp.sc.schedulerState.Persist(); err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable persist cluster data: ", err)
		log.Error("Unable persist cluster data: ", err)
		return
	}
	w.WriteHeader(200)
	fmt.Fprintf(w, "Success!")
}
func (schttp *SchedulerHTTPServer) getConfig(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)

	} else {
		w.WriteHeader(200)
		fmt.Fprint(w, cluster.RiakConfig)
	}
}

func (schttp *SchedulerHTTPServer) setAdvancedConfig(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read file: ", err)
		return
	}
	cluster.AdvancedConfig = string(data)
	if err := schttp.sc.schedulerState.Persist(); err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable persist cluster data: ", err)
		log.Error("Unable persist cluster data: ", err)
		return
	}
	w.WriteHeader(200)
	fmt.Fprint(w, "Success!")
}

func (schttp *SchedulerHTTPServer) getAdvancedConfig(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	w.WriteHeader(200)
	fmt.Fprint(w, cluster.AdvancedConfig)
}

func (schttp *SchedulerHTTPServer) serveClusters(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	json.NewEncoder(w).Encode(schttp.sc.schedulerState.Clusters)
}

func (schttp *SchedulerHTTPServer) getCluster(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]

	if !assigned {
		http.NotFound(w, r)
	} else {
		json.NewEncoder(w).Encode(cluster)
	}

}

func (schttp *SchedulerHTTPServer) removeNode(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeID := vars["node"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
	} else {
		node, assigned := cluster.Nodes[nodeID]
		if !assigned {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Node %s not found", nodeID)
		} else {
			node.KillNext()
			schttp.sc.schedulerState.Persist()
			w.WriteHeader(202)
		}
	}
}

func (schttp *SchedulerHTTPServer) createNode(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
	} else {
		node := cluster.CreateNode(schttp.sc)
		schttp.sc.schedulerState.Persist()
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(node)
	}
}

func (schttp *SchedulerHTTPServer) serveNodes(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
	} else {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(cluster.Nodes)
	}
}

func (schttp *SchedulerHTTPServer) nodeAAE(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeName := vars["node"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	node, assigned := cluster.Nodes[nodeName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Node %s not found", nodeName)
		return
	}
	rexHostname := fmt.Sprintf("%s:%d", node.Hostname, node.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	body, err := rexc.GetAAEStatusJSON(node.TaskData.FullyQualifiedNodeName)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read data: ", err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, body)
}
func (schttp *SchedulerHTTPServer) nodeStatus(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeName := vars["node"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	node, assigned := cluster.Nodes[nodeName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Node %s not found", nodeName)
		return
	}
	rexHostname := fmt.Sprintf("%s:%d", node.Hostname, node.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	body, err := rexc.GetStatusJSON(node.TaskData.FullyQualifiedNodeName)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read data: ", err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, body)
}
func (schttp *SchedulerHTTPServer) nodeRingready(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeName := vars["node"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	node, assigned := cluster.Nodes[nodeName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Node %s not found", nodeName)
		return
	}
	rexHostname := fmt.Sprintf("%s:%d", node.Hostname, node.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	body, err := rexc.GetRingreadyJSON(node.TaskData.FullyQualifiedNodeName)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read data: ", err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, body)
}
func (schttp *SchedulerHTTPServer) nodeTransfers(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeName := vars["node"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	node, assigned := cluster.Nodes[nodeName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Node %s not found", nodeName)
		return
	}
	rexHostname := fmt.Sprintf("%s:%d", node.Hostname, node.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	body, err := rexc.GetTransfersJSON(node.TaskData.FullyQualifiedNodeName)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read data: ", err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, body)
}
func (schttp *SchedulerHTTPServer) nodeTypes(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeName := vars["node"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	node, assigned := cluster.Nodes[nodeName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Node %s not found", nodeName)
		return
	}
	rexHostname := fmt.Sprintf("%s:%d", node.Hostname, node.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	body, err := rexc.GetBucketTypesJSON(node.TaskData.FullyQualifiedNodeName)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read data: ", err)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, body)
}
func (schttp *SchedulerHTTPServer) nodeCreateType(w http.ResponseWriter, r *http.Request) {
	schttp.sc.lock.Lock()
	defer schttp.sc.lock.Unlock()
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	nodeName := vars["node"]
	bucketType := vars["buckettype"]
	cluster, assigned := schttp.sc.schedulerState.Clusters[clusterName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Cluster %s not found", clusterName)
		return
	}
	node, assigned := cluster.Nodes[nodeName]
	if !assigned {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Node %s not found", nodeName)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to read data: ", err)
		return
	}
	rexHostname := fmt.Sprintf("%s:%d", node.Hostname, node.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)

	if _, err := rexc.CreateBucketTypeJSON(node.TaskData.FullyQualifiedNodeName, bucketType, string(data)); err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to create bucket type: ", err)
		log.Error("Unable to create bucket type: ", err)
		return
	}
	w.WriteHeader(204)
	fmt.Fprintf(w, "Created and activated bucket type %s", bucketType)
}

type simpleNode struct {
	host     string
	httpPort int64
	pbPort   int64
}

func (schttp *SchedulerHTTPServer) GetURI() string {
	return schttp.URI
}

func (schttp *SchedulerHTTPServer) healthcheck(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintln(w, "Scheduler: OK")
	w.WriteHeader(200)

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

	hostURI := fmt.Sprintf("http://%s:%d/static/executor_linux_amd64", hostname, port)
	riakURI := fmt.Sprintf("http://%s:%d/static/riak_linux_amd64.tar.gz", hostname, port)
	URI := fmt.Sprintf("http://%s:%d", hostname, port)
	//Info.Printf("Hosting artifact '%s' at '%s'", path, hostURI)
	log.Println("Serving at HostURI: ", hostURI)

	router := mux.NewRouter().StrictSlash(true)

	fs := http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""})

	// This rewrites /static/FOO -> FOO
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	fs2 := http.FileServer(&assetfs.AssetFS{Asset: artifacts.Asset, AssetDir: artifacts.AssetDir, Prefix: ""})
	router.PathPrefix("/static2/").Handler(http.StripPrefix("/static2/", fs2))

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

	router.HandleFunc("/api/v1/clusters", schttp.serveClusters)
	router.Methods("POST", "PUT").Path("/api/v1/clusters/{cluster}").HandlerFunc(schttp.createCluster)
	router.Methods("POST").Path("/api/v1/clusters/{cluster}/restart").HandlerFunc(schttp.restartCluster)
	router.Methods("DELETE").Path("/api/v1/clusters/{cluster}").HandlerFunc(schttp.removeCluster)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}").HandlerFunc(schttp.getCluster)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/nodes").HandlerFunc(schttp.serveNodes)
	router.Methods("DELETE").Path("/api/v1/clusters/{cluster}/nodes/{node}").HandlerFunc(schttp.removeNode)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/nodes/{node}/aae").HandlerFunc(schttp.nodeAAE)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/nodes/{node}/status").HandlerFunc(schttp.nodeStatus)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/nodes/{node}/ringready").HandlerFunc(schttp.nodeRingready)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/nodes/{node}/transfers").HandlerFunc(schttp.nodeTransfers)
	router.Methods("POST").Path("/api/v1/clusters/{cluster}/nodes/{node}/types/{buckettype}").HandlerFunc(schttp.nodeCreateType)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/nodes/{node}/types").HandlerFunc(schttp.nodeTypes)

	router.Methods("POST").Path("/api/v1/clusters/{cluster}/config").HandlerFunc(schttp.setConfig)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/config").HandlerFunc(schttp.getConfig)
	router.Methods("POST").Path("/api/v1/clusters/{cluster}/advancedConfig").HandlerFunc(schttp.setAdvancedConfig)
	router.Methods("GET").Path("/api/v1/clusters/{cluster}/advancedConfig").HandlerFunc(schttp.getAdvancedConfig)

	router.Methods("POST").Path("/api/v1/clusters/{cluster}/nodes").HandlerFunc(schttp.createNode)
	router.Methods("GET").Path("/healthcheck").HandlerFunc(schttp.healthcheck)

	// TODO: Add a function handler for /
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
