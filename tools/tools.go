package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/kr/pretty"
	"github.com/samuel/go-zookeeper/zk"
	"io"
	"time"
)

var (
	zookeeperAddr string
	nodes         int
	clusterName   string
	frameworkName string
	cmd           string
	client        *SchedulerHTTPClient
	config        string
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.IntVar(&nodes, "nodes", 1, "Nodes in new cluster")
	flag.StringVar(&clusterName, "cluster-name", "", "Name of new cluster")
	flag.StringVar(&config, "config", "", "filename of new config")

	flag.StringVar(&frameworkName, "name", "riakMesosFramework", "Framework Instance ID")
	flag.StringVar(&cmd, "command", "get-url",
		"get-url, get-clusters, get-cluster, create-cluster, "+
			"delete-cluster, get-nodes, add-node, add-nodes, get-state, "+
			"get-config, set-config, get-advanced-config, set-advanced-config")
	flag.Parse()

	if cmd == "" {
		fmt.Println("Please specify command")
		os.Exit(1)
	}
	log.SetLevel(log.DebugLevel)
}

func main() {
	switch cmd {
	case "get-url":
		fmt.Println(getURL())
	case "get-state":
		getState()
	case "delete-framework":
		respond(deleteFramework(), nil)
	case "zk-list-children":
		respondList(zkListChildren(), nil)
	case "zk-get-data":
		respond(zkGetData(), nil)
	case "zk-delete":
		respond("ok", zkDelete())
	case "get-clusters":
		client = NewSchedulerHTTPClient(getURL())
		respond(client.GetClusters())
	case "get-cluster":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetCluster(clusterName))
	case "create-cluster":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.CreateCluster(clusterName))
	case "delete-cluster":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.DeleteCluster(clusterName))
	case "get-nodes":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetNodes(clusterName))
	case "get-node-hosts":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetNodeHosts(clusterName))
	case "add-node":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.AddNode(clusterName))
	case "add-nodes":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		for i := 1; i <= nodes; i++ {
			respond(client.AddNode(clusterName))
		}
	case "get-config":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetClusterConfig(clusterName))
	case "set-config":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		client.SetClusterConfig(clusterName, getConfigData())
	case "get-advanced-config":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetClusterAdvancedConfig(clusterName))

	case "set-advanced-config":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		client.SetClusterAdvancedConfig(clusterName, getConfigData())
	default:
		log.Fatal("Unknown command")
	}
}

func createBucketType(name string, props string) {
	// {
	//    "17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c":{
	//       "UUID":"17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c",
	//       "DestinationState":2,
	//       "CurrentState":2,
	//       "TaskStatus":{
	//          "task_id":{
	//             "value":"riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1"
	//          },
	//          "state":1,
	//          "message":"Reconciliation: Latest task state",
	//          "source":0,
	//          "reason":9,
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "timestamp":1.444799783110596e+09
	//       },
	//       "Generation":1,
	//       "LastTaskInfo":{
	//          "name":"riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1",
	//          "task_id":{
	//             "value":"riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1"
	//          },
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "resources":[
	//             {
	//                "name":"cpus",
	//                "type":0,
	//                "scalar":{
	//                   "value":0.3
	//                }
	//             },
	//             {
	//                "name":"ports",
	//                "type":1,
	//                "ranges":{
	//                   "range":[
	//                      {
	//                         "begin":31101,
	//                         "end":31110
	//                      }
	//                   ]
	//                }
	//             },
	//             {
	//                "name":"mem",
	//                "type":0,
	//                "scalar":{
	//                   "value":320
	//                }
	//             }
	//          ],
	//          "executor":{
	//             "executor_id":{
	//                "value":"riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1"
	//             },
	//             "framework_id":{
	//                "value":"20151007-201948-2486378412-5050-1302-0005"
	//             },
	//             "command":{
	//                "uris":[
	//                   {
	//                      "value":"http://ip-172-31-51-148:33076/static/executor_linux_amd64",
	//                      "executable":true
	//                   }
	//                ],
	//                "shell":false,
	//                "value":"./executor_linux_amd64",
	//                "arguments":[
	//                   "./executor_linux_amd64",
	//                   "-logtostderr=true",
	//                   "-taskinfo",
	//                   "riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1"
	//                ]
	//             },
	//             "resources":[
	//                {
	//                   "name":"cpus",
	//                   "type":0,
	//                   "scalar":{
	//                      "value":0.01
	//                   }
	//                },
	//                {
	//                   "name":"mem",
	//                   "type":0,
	//                   "scalar":{
	//                      "value":32
	//                   }
	//                }
	//             ],
	//             "name":"Executor (Go)",
	//             "source":"Riak Mesos Framework (Go)"
	//          },
	//          "data":"eyJGdWxseVF1YWxpZmllZE5vZGVOYW1lIjoicmlhay1teWNsdXN0ZXItMTdjN2I4NDgtN2ZiZS00ZGM1LTk1YzEtYzQ4YzgyY2Q4ZjVjLTFAaXAtMTcyLTMxLTUxLTE0OC5lYzIuaW50ZXJuYWwiLCJab29rZWVwZXJzIjpbImxvY2FsaG9zdDoyMTgxIl0sIk5vZGVJRCI6IjE3YzdiODQ4LTdmYmUtNGRjNS05NWMxLWM0OGM4MmNkOGY1YyIsIkZyYW1ld29ya05hbWUiOiJyaWFrIiwiQ2x1c3Rlck5hbWUiOiJteWNsdXN0ZXIiLCJVUkkiOiJodHRwOi8vaXAtMTcyLTMxLTUxLTE0ODozMzA3NiIsIlVzZVN1cGVyQ2hyb290Ijp0cnVlLCJIVFRQUG9ydCI6MzExMDEsIlBCUG9ydCI6MzExMDIsIkhhbmRvZmZQb3J0IjowLCJEaXN0ZXJsUG9ydCI6MzExMDN9"
	//       },
	//       "LastOfferUsed":{
	//          "id":{
	//             "value":"20151007-201948-2486378412-5050-1302-O6357"
	//          },
	//          "framework_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-0005"
	//          },
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "hostname":"ip-172-31-51-148.ec2.internal",
	//          "url":{
	//             "scheme":"http",
	//             "address":{
	//                "hostname":"ip-172-31-51-148.ec2.internal",
	//                "ip":"172.31.51.148",
	//                "port":5051
	//             },
	//             "path":"/slave(1)"
	//          },
	//          "resources":[
	//             {
	//                "name":"cpus",
	//                "type":0,
	//                "scalar":{
	//                   "value":8
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"mem",
	//                "type":0,
	//                "scalar":{
	//                   "value":14015
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"disk",
	//                "type":0,
	//                "scalar":{
	//                   "value":75375
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"ports",
	//                "type":1,
	//                "ranges":{
	//                   "range":[
	//                      {
	//                         "begin":31000,
	//                         "end":32000
	//                      }
	//                   ]
	//                },
	//                "role":"*"
	//             }
	//          ]
	//       },
	//       "TaskData":{
	//          "FullyQualifiedNodeName":"riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1@ip-172-31-51-148.ec2.internal",
	//          "Zookeepers":[
	//             "localhost:2181"
	//          ],
	//          "NodeID":"17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c",
	//          "FrameworkName":"riak",
	//          "ClusterName":"mycluster",
	//          "URI":"http://ip-172-31-51-148:33076",
	//          "UseSuperChroot":true,
	//          "HTTPPort":31101,
	//          "PBPort":31102,
	//          "HandoffPort":0,
	//          "DisterlPort":31103
	//       },
	//       "FrameworkName":"riak",
	//       "ClusterName":"mycluster"
	//    },
	//    "2c0045d8-dc95-496c-a879-965701ded919":{
	//       "UUID":"2c0045d8-dc95-496c-a879-965701ded919",
	//       "DestinationState":2,
	//       "CurrentState":2,
	//       "TaskStatus":{
	//          "task_id":{
	//             "value":"riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1"
	//          },
	//          "state":1,
	//          "message":"Reconciliation: Latest task state",
	//          "source":0,
	//          "reason":9,
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "timestamp":1.44479978311052e+09
	//       },
	//       "Generation":1,
	//       "LastTaskInfo":{
	//          "name":"riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1",
	//          "task_id":{
	//             "value":"riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1"
	//          },
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "resources":[
	//             {
	//                "name":"cpus",
	//                "type":0,
	//                "scalar":{
	//                   "value":0.3
	//                }
	//             },
	//             {
	//                "name":"ports",
	//                "type":1,
	//                "ranges":{
	//                   "range":[
	//                      {
	//                         "begin":31376,
	//                         "end":31385
	//                      }
	//                   ]
	//                }
	//             },
	//             {
	//                "name":"mem",
	//                "type":0,
	//                "scalar":{
	//                   "value":320
	//                }
	//             }
	//          ],
	//          "executor":{
	//             "executor_id":{
	//                "value":"riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1"
	//             },
	//             "framework_id":{
	//                "value":"20151007-201948-2486378412-5050-1302-0005"
	//             },
	//             "command":{
	//                "uris":[
	//                   {
	//                      "value":"http://ip-172-31-51-148:33076/static/executor_linux_amd64",
	//                      "executable":true
	//                   }
	//                ],
	//                "shell":false,
	//                "value":"./executor_linux_amd64",
	//                "arguments":[
	//                   "./executor_linux_amd64",
	//                   "-logtostderr=true",
	//                   "-taskinfo",
	//                   "riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1"
	//                ]
	//             },
	//             "resources":[
	//                {
	//                   "name":"cpus",
	//                   "type":0,
	//                   "scalar":{
	//                      "value":0.01
	//                   }
	//                },
	//                {
	//                   "name":"mem",
	//                   "type":0,
	//                   "scalar":{
	//                      "value":32
	//                   }
	//                }
	//             ],
	//             "name":"Executor (Go)",
	//             "source":"Riak Mesos Framework (Go)"
	//          },
	//          "data":"eyJGdWxseVF1YWxpZmllZE5vZGVOYW1lIjoicmlhay1teWNsdXN0ZXItMmMwMDQ1ZDgtZGM5NS00OTZjLWE4NzktOTY1NzAxZGVkOTE5LTFAaXAtMTcyLTMxLTUxLTE0OC5lYzIuaW50ZXJuYWwiLCJab29rZWVwZXJzIjpbImxvY2FsaG9zdDoyMTgxIl0sIk5vZGVJRCI6IjJjMDA0NWQ4LWRjOTUtNDk2Yy1hODc5LTk2NTcwMWRlZDkxOSIsIkZyYW1ld29ya05hbWUiOiJyaWFrIiwiQ2x1c3Rlck5hbWUiOiJteWNsdXN0ZXIiLCJVUkkiOiJodHRwOi8vaXAtMTcyLTMxLTUxLTE0ODozMzA3NiIsIlVzZVN1cGVyQ2hyb290Ijp0cnVlLCJIVFRQUG9ydCI6MzEzNzYsIlBCUG9ydCI6MzEzNzcsIkhhbmRvZmZQb3J0IjowLCJEaXN0ZXJsUG9ydCI6MzEzNzh9"
	//       },
	//       "LastOfferUsed":{
	//          "id":{
	//             "value":"20151007-201948-2486378412-5050-1302-O6357"
	//          },
	//          "framework_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-0005"
	//          },
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "hostname":"ip-172-31-51-148.ec2.internal",
	//          "url":{
	//             "scheme":"http",
	//             "address":{
	//                "hostname":"ip-172-31-51-148.ec2.internal",
	//                "ip":"172.31.51.148",
	//                "port":5051
	//             },
	//             "path":"/slave(1)"
	//          },
	//          "resources":[
	//             {
	//                "name":"cpus",
	//                "type":0,
	//                "scalar":{
	//                   "value":8
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"mem",
	//                "type":0,
	//                "scalar":{
	//                   "value":14015
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"disk",
	//                "type":0,
	//                "scalar":{
	//                   "value":75375
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"ports",
	//                "type":1,
	//                "ranges":{
	//                   "range":[
	//                      {
	//                         "begin":31000,
	//                         "end":32000
	//                      }
	//                   ]
	//                },
	//                "role":"*"
	//             }
	//          ]
	//       },
	//       "TaskData":{
	//          "FullyQualifiedNodeName":"riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1@ip-172-31-51-148.ec2.internal",
	//          "Zookeepers":[
	//             "localhost:2181"
	//          ],
	//          "NodeID":"2c0045d8-dc95-496c-a879-965701ded919",
	//          "FrameworkName":"riak",
	//          "ClusterName":"mycluster",
	//          "URI":"http://ip-172-31-51-148:33076",
	//          "UseSuperChroot":true,
	//          "HTTPPort":31376,
	//          "PBPort":31377,
	//          "HandoffPort":0,
	//          "DisterlPort":31378
	//       },
	//       "FrameworkName":"riak",
	//       "ClusterName":"mycluster"
	//    },
	//    "555537e2-a7f1-47f2-a411-b3b926196915":{
	//       "UUID":"555537e2-a7f1-47f2-a411-b3b926196915",
	//       "DestinationState":2,
	//       "CurrentState":2,
	//       "TaskStatus":{
	//          "task_id":{
	//             "value":"riak-mycluster-555537e2-a7f1-47f2-a411-b3b926196915-1"
	//          },
	//          "state":1,
	//          "message":"Reconciliation: Latest task state",
	//          "source":0,
	//          "reason":9,
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "timestamp":1.444799783110576e+09
	//       },
	//       "Generation":1,
	//       "LastTaskInfo":{
	//          "name":"riak-mycluster-555537e2-a7f1-47f2-a411-b3b926196915-1",
	//          "task_id":{
	//             "value":"riak-mycluster-555537e2-a7f1-47f2-a411-b3b926196915-1"
	//          },
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "resources":[
	//             {
	//                "name":"cpus",
	//                "type":0,
	//                "scalar":{
	//                   "value":0.3
	//                }
	//             },
	//             {
	//                "name":"ports",
	//                "type":1,
	//                "ranges":{
	//                   "range":[
	//                      {
	//                         "begin":31337,
	//                         "end":31346
	//                      }
	//                   ]
	//                }
	//             },
	//             {
	//                "name":"mem",
	//                "type":0,
	//                "scalar":{
	//                   "value":320
	//                }
	//             }
	//          ],
	//          "executor":{
	//             "executor_id":{
	//                "value":"riak-mycluster-555537e2-a7f1-47f2-a411-b3b926196915-1"
	//             },
	//             "framework_id":{
	//                "value":"20151007-201948-2486378412-5050-1302-0005"
	//             },
	//             "command":{
	//                "uris":[
	//                   {
	//                      "value":"http://ip-172-31-51-148:33076/static/executor_linux_amd64",
	//                      "executable":true
	//                   }
	//                ],
	//                "shell":false,
	//                "value":"./executor_linux_amd64",
	//                "arguments":[
	//                   "./executor_linux_amd64",
	//                   "-logtostderr=true",
	//                   "-taskinfo",
	//                   "riak-mycluster-555537e2-a7f1-47f2-a411-b3b926196915-1"
	//                ]
	//             },
	//             "resources":[
	//                {
	//                   "name":"cpus",
	//                   "type":0,
	//                   "scalar":{
	//                      "value":0.01
	//                   }
	//                },
	//                {
	//                   "name":"mem",
	//                   "type":0,
	//                   "scalar":{
	//                      "value":32
	//                   }
	//                }
	//             ],
	//             "name":"Executor (Go)",
	//             "source":"Riak Mesos Framework (Go)"
	//          },
	//          "data":"eyJGdWxseVF1YWxpZmllZE5vZGVOYW1lIjoicmlhay1teWNsdXN0ZXItNTU1NTM3ZTItYTdmMS00N2YyLWE0MTEtYjNiOTI2MTk2OTE1LTFAaXAtMTcyLTMxLTUxLTE0OC5lYzIuaW50ZXJuYWwiLCJab29rZWVwZXJzIjpbImxvY2FsaG9zdDoyMTgxIl0sIk5vZGVJRCI6IjU1NTUzN2UyLWE3ZjEtNDdmMi1hNDExLWIzYjkyNjE5NjkxNSIsIkZyYW1ld29ya05hbWUiOiJyaWFrIiwiQ2x1c3Rlck5hbWUiOiJteWNsdXN0ZXIiLCJVUkkiOiJodHRwOi8vaXAtMTcyLTMxLTUxLTE0ODozMzA3NiIsIlVzZVN1cGVyQ2hyb290Ijp0cnVlLCJIVFRQUG9ydCI6MzEzMzcsIlBCUG9ydCI6MzEzMzgsIkhhbmRvZmZQb3J0IjowLCJEaXN0ZXJsUG9ydCI6MzEzMzl9"
	//       },
	//       "LastOfferUsed":{
	//          "id":{
	//             "value":"20151007-201948-2486378412-5050-1302-O6358"
	//          },
	//          "framework_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-0005"
	//          },
	//          "slave_id":{
	//             "value":"20151007-201948-2486378412-5050-1302-S0"
	//          },
	//          "hostname":"ip-172-31-51-148.ec2.internal",
	//          "url":{
	//             "scheme":"http",
	//             "address":{
	//                "hostname":"ip-172-31-51-148.ec2.internal",
	//                "ip":"172.31.51.148",
	//                "port":5051
	//             },
	//             "path":"/slave(1)"
	//          },
	//          "resources":[
	//             {
	//                "name":"cpus",
	//                "type":0,
	//                "scalar":{
	//                   "value":7.380000000000001
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"mem",
	//                "type":0,
	//                "scalar":{
	//                   "value":13311
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"disk",
	//                "type":0,
	//                "scalar":{
	//                   "value":75375
	//                },
	//                "role":"*"
	//             },
	//             {
	//                "name":"ports",
	//                "type":1,
	//                "ranges":{
	//                   "range":[
	//                      {
	//                         "begin":31000,
	//                         "end":31100
	//                      },
	//                      {
	//                         "begin":31111,
	//                         "end":31375
	//                      },
	//                      {
	//                         "begin":31386,
	//                         "end":32000
	//                      }
	//                   ]
	//                },
	//                "role":"*"
	//             }
	//          ],
	//          "executor_ids":[
	//             {
	//                "value":"riak-mycluster-2c0045d8-dc95-496c-a879-965701ded919-1"
	//             },
	//             {
	//                "value":"riak-mycluster-17c7b848-7fbe-4dc5-95c1-c48c82cd8f5c-1"
	//             }
	//          ]
	//       },
	//       "TaskData":{
	//          "FullyQualifiedNodeName":"riak-mycluster-555537e2-a7f1-47f2-a411-b3b926196915-1@ip-172-31-51-148.ec2.internal",
	//          "Zookeepers":[
	//             "localhost:2181"
	//          ],
	//          "NodeID":"555537e2-a7f1-47f2-a411-b3b926196915",
	//          "FrameworkName":"riak",
	//          "ClusterName":"mycluster",
	//          "URI":"http://ip-172-31-51-148:33076",
	//          "UseSuperChroot":true,
	//          "HTTPPort":31337,
	//          "PBPort":31338,
	//          "HandoffPort":0,
	//          "DisterlPort":31339
	//       },
	//       "FrameworkName":"riak",
	//       "ClusterName":"mycluster"
	//    }
	// }
}

func respondList(val []string, err error) {
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(val)
}

func respond(val string, err error) {
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(val)
}

func getURL() string {
	overrideURL := os.Getenv("RM_API")

	if overrideURL != "" {
		return overrideURL
	}

	mgr := metadata_manager.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	zkNode, err := mgr.GetRootNode().GetChild("uri")
	if err != nil {
		log.Panic(err)
	}
	return string(zkNode.GetData())
}

func deleteFramework() string {
	frameworkName = "/riak/frameworks/" + frameworkName
	zkDelete()
	return "ok"
}

func zkListChildren() []string {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		log.Panic(err)
	}
	children, _, err := conn.Children(frameworkName)

	if err != nil {
		log.Panic(err)
	}
	return children
}

func zkGetData() string {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		log.Panic(err)
	}
	data, _, err := conn.Get(frameworkName)

	if err != nil {
		log.Panic(err)
	}
	return string(data)
}

func zkDelete() error {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		log.Panic(err)
	}

	zkDeleteChildren(conn, frameworkName)

	return nil
}

func getConfigData() io.Reader {
	if config == "" {
		fmt.Println("Please specify value for configuration file name")
		os.Exit(1)
	}
	data, err := os.Open(config)
	if err != nil {
		fmt.Println("Error while retrieving configuration file: ", err)
		os.Exit(2)
	}
	return data
}

func requireClusterName() {
	if clusterName == "" {
		fmt.Println("Please specify value for cluster name")
		os.Exit(1)
	}
}

func zkDeleteChildren(conn *zk.Conn, path string) {
	children, _, _ := conn.Children(path)

	// Leaf
	if len(children) == 0 {
		fmt.Println("Deleting ", path)
		err := conn.Delete(path, -1)
		if err != nil {
			log.Panic(err)
		}
		return
	}

	// Branches
	for _, name := range children {
		zkDeleteChildren(conn, path+"/"+name)
	}

	return
}

func getState() {
	mm := metadata_manager.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	zkNode, err := mm.GetRootNode().GetChild("SchedulerState")
	if err != zk.ErrNoNode {
		// This results in the inclusion of all of the bindata used for scheduler... Lets not deserialize
		//ss, err := scheduler.DeserializeSchedulerState(zkNode.GetData())
		ss := zkNode.GetData()
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("%# v", pretty.Formatter(ss))

	}
}
