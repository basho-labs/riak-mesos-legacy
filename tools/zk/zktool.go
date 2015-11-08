package main

import (
	"flag"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"os"
	"time"
)

var (
	zookeeperAddr string
	cmd           string
	frameworkName string
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&frameworkName, "name", "riakMesosFramework", "Framework Instance ID")
	flag.StringVar(&cmd, "command", "get-url",
		"get-url, zk-list-children, zk-get-data, zk-delete")
	flag.Parse()

	if cmd == "" {
		fmt.Println("Please specify command")
		os.Exit(1)
	}
}

func main() {
	switch cmd {
	case "get-url":
		fmt.Println(getURL())
	case "zk-list-children":
		respondList(zkListChildren(), nil)
	case "zk-get-data":
		respond(zkGetData(), nil)
	case "zk-delete":
		respond("ok", zkDelete())
	default:
		fmt.Println("Unknown command")
	}
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
	frameworkName = "/riak/frameworks/" + frameworkName + "/uri"
	return zkGetData()
}

func deleteFramework() string {
	frameworkName = "/riak/frameworks/" + frameworkName
	zkDelete()
	return "ok"
}

func zkListChildren() []string {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		fmt.Println(err)
	}
	children, _, err := conn.Children(frameworkName)

	if err != nil {
		fmt.Println(err)
	}
	return children
}

func zkGetData() string {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		fmt.Println(err)
	}
	data, _, err := conn.Get(frameworkName)

	if err != nil {
		fmt.Println(err)
	}
	return string(data)
}

func zkDelete() error {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		fmt.Println(err)
	}

	zkDeleteChildren(conn, frameworkName)

	return nil
}

func zkDeleteChildren(conn *zk.Conn, path string) {
	children, _, _ := conn.Children(path)

	// Leaf
	if len(children) == 0 {
		fmt.Println("Deleting ", path)
		err := conn.Delete(path, -1)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	// Branches
	for _, name := range children {
		zkDeleteChildren(conn, path+"/"+name)
	}

	return
}
