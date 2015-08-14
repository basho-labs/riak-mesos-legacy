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
from dcos import marathon, util
from dcos_riak import constants


def api_url(name):
    if name == "":
        name="riak"

    client = marathon.create_client()
    tasks = client.get_tasks(name)

    if len(tasks) == 0:
        usage()
        print("\nTry running the following to verify that "+ name + " is the name \nof your service instance:\n")
        print("    dcos service\n")
        raise CliError("Riak is not running.")

    base_url = util.get_config().get('core.dcos_url').rstrip("/")
    return base_url + "/service/" + name + "/"

def usage():
    print("dcos riak <subcommand> [<options>]")
    print("Subcommands: ")
    print("    --get-clusters")
    print("    --get-nodes <cluster-name>")
    print("    --get-nodes <cluster-name> <node>")
    print("    --create-cluster <cluster-name>")
    print("    --add-node <cluster-name>")
    print("    --info")
    print("    --version")
    print("Options: ")
    print("    --framework-name <framework-name>")
    print("    --debug")

def maybe_debug(flag, r):
    if flag == 1:
        print("[DEBUG]\n")
        print("Status: "+ str(r.status_code))
        print("Text: "+ r.text)
        print("[/DEBUG]")

def run(args):
    help_arg = len(args) > 0 and args[0] == "help"
    if help_arg:
        usage()
        return 0

    service_url = ""
    flag = 0

    if "--debug" in args:
        flag = 1

    if "--framework-name" in args and args.index('--framework-name')+1 < len(args):
        service_url = api_url(args[args.index('--framework-name')+1])
        if args.index('--framework-name')+2 == len(args):
            print("Service URL: " + service_url)
    else:
        service_url = api_url("")

    for i, opt in enumerate(args):
        if opt == "--get-cluster":
            if i+1 < len(args):
                r = requests.get(service_url + "clusters/" + args[i+1])
                maybe_debug(flag, r)
                if r.status_code == 200:
                    print("Cluster: "+ r.text, end="\n")
                    r = requests.get(service_url + "clusters/" + args[i+1] + "/nodes")
                    maybe_debug(flag, r)
                    try:
                        nodes = json.loads(r.text)
                        print("Nodes: [" + ', '.join(nodes.keys()) + "]", end="")
                        break
                    except:
                        print("Nodes: []")
                else:
                    print("Cluster not created")
            else:
                usage()

        if opt == "--get-clusters":
            r = requests.get(service_url + "clusters")
            maybe_debug(flag, r)
            if r.status_code == 200:
                try:
                    clusters = json.loads(r.text)
                    print("Clusters: [" + ', '.join(clusters.keys()) + "]", end="")
                    break
                except:
                    print("[]")
            else:
                print("No clusters created")

        if opt == "--get-nodes":
            if i+1 < len(args):
                r = requests.get(service_url + "clusters/" + args[i+1] + "/nodes")
                maybe_debug(flag, r)
                try:
                    nodes = json.loads(r.text)
                    print("Nodes: [" + ', '.join(nodes.keys()) + "]", end="")
                    break
                except:
                    print("Nodes: []")
            else:
                usage()

        if opt == "--create-cluster":
            if i+1 < len(args):
                r = requests.post(service_url + "clusters/" + args[i+1], data="")
                maybe_debug(flag, r)
                if r.text == "" or r.status_code != 200:
                    print("Cluster already exists")
                else:
                    print("Created cluster: "+ r.text, end="\n")
            else:
                usage()

        if opt == "--add-node":
            if i+1 < len(args):
                r = requests.post(service_url + "clusters/" + args[i+1] + "/nodes", data="")
                maybe_debug(flag, r)
                try:
                    node = json.loads(r.text)
                    print("Added node: " + node["UUID"], end="")
                    break
                except:
                    print("Error adding node.")
            else:
                usage()

    print("")

    return 0


class CliError(Exception):
    pass


def main():
    args = sys.argv[2:]  # remove dcos-riak & riak
    if len(args) == 0:
        usage()
        return 0

    if len(args) == 1 and args[0] == "--info":
        print("Start and manage Riak nodes")
        return 0

    if len(args) == 1 and args[0] == "--version":
        print(constants.version)
        return 0

    if len(args) == 1 and args[0] == "--config-schema":
        print("{}")
        return 0

    if "--help" in args or "-h" in args:
        if "--help" in args:
            args.remove("--help")

        if "-h" in args:
            args.remove("-h")

        args.insert(0, "help")

    try:
        return run(args)
    except CliError as e:
        print("Error: " + str(e), file=sys.stderr)
        return 1
