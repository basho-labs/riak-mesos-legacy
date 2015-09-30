#
#    Copyright (C) 2015 Basho Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""DCOS Riak"""
from __future__ import print_function

import os
import subprocess
import sys
import requests
import json

import pkg_resources
from dcos import marathon, util, errors
from dcos_riak import constants

def usage():
    print("Command line utility for the Riak Mesos Framework / DCOS Service.")
    print("This utility provides tools for modifying and accessing your Riak")
    print("on Mesos installation.")
    print("")
    print("Usage: dcos riak <subcommands> [options]")
    print("")
    print("Subcommands: ")
    print("    cluster list")
    print("    cluster create")
    print("    node list")
    print("    node add [--nodes <number>]")
    print("    proxy config [--zk <host:port> [--os centos|coreos|ubuntu][--disable-super-chroot]]")
    print("    proxy install [--zk <host:port> [--os centos|coreos|ubuntu][--disable-super-chroot]]")
    print("    proxy uninstall")
    print("    proxy endpoints [--public-dns <host>]")
    print("")
    print("Options (available on most commands): ")
    print("    --cluster <cluster-name>      Default: riak-cluster")
    print("    --framework <framework-name>  Default: riak")
    print("    --debug")
    print("    --help")
    print("    --info")
    print("    --version")
    print("")

def api_url(framework):
    client = marathon.create_client()
    tasks = client.get_tasks(framework)

    if len(tasks) == 0:
        usage()
        print("\nTry running the following to verify that "+ framework + " is the name \nof your service instance:\n")
        print("    dcos service\n")
        raise CliError("Riak is not running, try with --framework <framework-name>.")

    base_url = util.get_config().get('core.dcos_url').rstrip("/")
    return base_url + "/service/" + framework + "/"

def format_json_array_keys(description, json_str, failure):
    try:
        obj_arr = json.loads(json_str)
        print(description + "[" + ', '.join(obj_arr.keys()) + "]", end="")
    except:
        print(description + failure)

def format_json_value(description, json_str, key, failure):
    try:
        obj = json.loads(json_str)
        print(description + obj[key], end="")
    except:
        print(description + failure)

def get_clusters(service_url, debug_flag):
    r = requests.get(service_url + "clusters")
    debug_request(debug_flag, r)
    if r.status_code == 200:
        format_json_array_keys("Clusters: ", r.text, "[]")
    else:
        print("No clusters created")

def get_cluster(service_url, name, debug_flag):
    r = requests.get(service_url + "clusters/" + name)
    debug_request(debug_flag, r)
    if r.status_code == 200:
        format_json_value("Cluster: ", r.text, "Name", "Error getting cluster.")
    else:
        print("Cluster not created.")

def create_cluster(service_url, name, debug_flag):
    r = requests.post(service_url + "clusters/" + name, data="")
    debug_request(debug_flag, r)
    if r.text == "" or r.status_code != 200:
        print("Cluster already exists")
    else:
        format_json_value("Added cluster: ", r.text, "Name", "Error creating cluster.")

def get_nodes(service_url, name, debug_flag):
    r = requests.get(service_url + "clusters/" + name + "/nodes")
    debug_request(debug_flag, r)
    format_json_array_keys("Nodes: ", r.text, "[]")

def add_node(service_url, name, debug_flag):
    r = requests.post(service_url + "clusters/" + name + "/nodes", data="")
    debug_request(debug_flag, r)
    if r.status_code == 404:
        print(r.text)
    else:
        format_json_value("New node: ", r.text, "UUID", "Error adding node.")
    print("")

def uninstall_director(framework):
    try:
        client = marathon.create_client()
        client.remove_app('/' + framework + '-director')
        print("Finished removing " + '/' + framework + '-director' + " from marathon")
    except errors.DCOSException as e:
        print(e.message)
        raise CliError("Unable to uninstall marathon app")
    except:
        raise CliError("Unable to communicate with marathon.")

def install_director(framework, cluster, zookeeper, op_sys, disable_super_chroot_flag):
    try:
        director_conf = generate_director_config(framework, cluster, zookeeper, op_sys, disable_super_chroot_flag)
        director_json = json.loads(director_conf)
        client = marathon.create_client()
        client.add_app(director_json)
        print("Finished adding " + director_json['id'] + " to marathon")
    except errors.DCOSException as e:
        print(e.message)
        raise CliError("Unable to create marathon app")
    except:
        raise CliError("Unable to communicate with marathon.")

def generate_director_config(framework, cluster, zookeeper, op_sys, disable_super_chroot_flag):
    try:

        framework_host = framework + ".marathon.mesos"
        client = marathon.create_client()
        app = client.get_app(framework)
        ports = app['ports']
        explorer_port = str(ports[1])
        return '{"id": "/' + framework + '-director","cmd": "./riak_mesos_director/director_linux_amd64","cpus": 0.5,"mem": 1024.0,"ports": [0, 0, 0, 0],"instances": 1,"constraints": [["hostname", "UNIQUE"]],"acceptedResourceRoles": ["slave_public"],"env": {"USE_SUPER_CHROOT": "'+ str(not disable_super_chroot_flag).lower() + '","FRAMEWORK_HOST": "' + framework_host + '","FRAMEWORK_PORT": "' + explorer_port + '","DIRECTOR_ZK": "' + zookeeper + '","DIRECTOR_FRAMEWORK": "' + framework + '","DIRECTOR_CLUSTER": "' + cluster + '"},"uris": ["http://riak-tools.s3.amazonaws.com/riak-mesos/' + op_sys + '/riak_mesos_director_linux_amd64_0.1.1.tar.gz"],"healthChecks": [{"protocol": "HTTP","path": "/health","gracePeriodSeconds": 3,"intervalSeconds": 10,"portIndex": 2,"timeoutSeconds": 10,"maxConsecutiveFailures": 3}]}'
    except errors.DCOSException as e:
        print(e.message)
        raise CliError("Unable to create marathon app")
    except:
        raise CliError("Unable to generate proxy config for framework: " + framework)

def get_director_urls(framework, public_dns):
    try:
        client = marathon.create_client()
        app = client.get_app(framework + '-director')
        ports = app['ports']
        print("\nLoad Balanced Riak Cluster (HTTP)")
        print("    http://" + public_dns + ":" + str(ports[0]))
        print("\nLoad Balanced Riak Cluster (Protobuf)")
        print("    http://" + public_dns + ":" + str(ports[1]))
        print("\nRiak Mesos Director API (HTTP)")
        print("    http://" + public_dns + ":" + str(ports[2]))
        print("\nRiak Explorer and API (HTTP)")
        print("    http://" + public_dns + ":" + str(ports[3]))
    except:
        raise CliError("Unable to get ports for: riak-director")

def validate_arg(opt, arg, arg_type = "string"):
    if arg.startswith("-"):
        raise CliError("Invalid argument for opt: " + opt + " [" + arg + "].")
    if arg_type == "integer" and not arg.isdigit():
        raise CliError("Invalid integer for opt: " + opt + " [" + arg + "].")

def extract_flag(args, name):
    val = False
    if name in args:
        index = args.index(name)
        val = True
        del args[index]
    return [args, val]

def extract_option(args, name, default, arg_type = "string"):
    val = default
    if name in args:
        index = args.index(name)
        if index+1 < len(args):
            val = args[index+1]
            validate_arg(name, val, arg_type)
            del args[index]
            del args[index]
        else:
            usage()
            print("")
            raise CliError("Not enough arguments for: " + name)
    return [args, val]

def debug_request(debug_flag, r):
    debug(debug_flag, "HTTP Status: "+ str(r.status_code))
    debug(debug_flag, "HTTP Response Text: "+ r.text)

def debug(debug_flag, debug_string):
    if debug_flag:
        print("[DEBUG]" + debug_string + "[/DEBUG]")

def run(args):
    args, help_flag = extract_flag(args, "--help")
    args, debug_flag = extract_flag(args, "--debug")
    args, disable_super_chroot_flag = extract_flag(args, "--disable-super-chroot")
    args, op_sys = extract_option(args, "--os", "coreos")
    args, framework = extract_option(args, "--framework", "riak")
    args, cluster = extract_option(args, "--cluster", "riak-cluster")
    args, zk = extract_option(args, "--zk", "master.mesos:2181")
    args, num_nodes = extract_option(args, "--nodes", "1", "integer")
    num_nodes = int(num_nodes)
    args, public_dns = extract_option(args, "--public-dns", "{{public.dns}}")

    cmd = ' '.join(args)

    debug(debug_flag, "Framework: " + framework)
    debug(debug_flag, "Cluster: " + cluster)
    debug(debug_flag, "Zookeeper: " + zk)
    debug(debug_flag, "Public DNS: " + public_dns)
    debug(debug_flag, "Nodes: " + str(num_nodes))
    debug(debug_flag, "Command: " + cmd)

    service_url = api_url(framework) + "api/v1/"
    debug(debug_flag, "Service URL: " + service_url)

    if cmd == "cluster list" or cmd == "cluster":
        if help_flag:
            print("Retrieves a list of cluster names")
            return 0
        else:
            get_clusters(service_url, debug_flag)
    elif cmd == "cluster create":
        if help_flag:
            print("Creates a new cluster. Secify the name with --cluster (default is riak-cluster).")
        else:
            create_cluster(service_url, cluster, debug_flag)
    elif cmd == "node list" or cmd == "node":
        if help_flag:
            print("Retrieves a list of node ids for a given --cluster (default is riak-cluster).")
        else:
            get_nodes(service_url, cluster, debug_flag)
    elif cmd == "node add":
        if help_flag:
            print("Adds one or more (using --nodes) nodes to a --cluster (default is riak-cluster).")
        else:
            for x in range(0, num_nodes):
                add_node(service_url, cluster, debug_flag)
    elif cmd == "proxy config" or cmd == "proxy":
        if help_flag:
            print("Generates a marathon json config using --zookeeper (default is master.mesos:2181) and --cluster (default is riak-cluster).")
        else:
            print(generate_director_config(framework, cluster, zk, op_sys, disable_super_chroot_flag))
    elif cmd == "proxy install":
        if help_flag:
            print("Installs a riak-mesos-director marathon app on the public Mesos node using --zookeeper (default is master.mesos:2181) and --cluster (default is riak-cluster).")
        else:
            install_director(framework, cluster, zk, op_sys, disable_super_chroot_flag)
    elif cmd == "proxy uninstall":
        if help_flag:
            print("Uninstalls the riak-mesos-director marathon app.")
        else:
            uninstall_director(framework)
    elif cmd == "proxy endpoints":
        if help_flag:
            print("Lists the endpoints exposed by a riak-mesos-director marathon app --public-dns (default is {{public-dns}}).")
        else:
            get_director_urls(framework, public_dns)
    elif cmd == "":
        print("No commands executed")
    elif cmd.startswith("-"):
        raise CliError("Unrecognized option: " + cmd)
    else:
        raise CliError("Unrecognized command: " + cmd)

    print("")

    return 0


class CliError(Exception):
    pass


def main():
    args = sys.argv[2:]  # remove dcos-riak & riak
    if len(args) == 0:
        usage()
        return 0

    if "--info" in args:
        print("Start and manage Riak nodes")
        return 0

    if "--version" in args:
        print(constants.version)
        return 0

    if "--config-schema" in args:
        print("{}")
        return 0

    try:
        return run(args)
    except CliError as e:
        print("Error: " + str(e), file=sys.stderr)
        return 1
