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

import pkg_resources
from dcos import marathon, util
from dcos_riak import constants


def api_url():
    client = marathon.create_client()
    tasks = client.get_tasks("riak")

    if len(tasks) == 0:
        raise CliError("Riak is not running")

    base_url = util.get_config().get('core.dcos_url').rstrip("/")
    return base_url + '/service/riak/'


def run(args):
    help_arg = len(args) > 0 and args[0] == "help"
    if help_arg:
        args[0] = "help"

    command = ["tools_linux_amd64"]# TODO decide whether to implement tools in python or include tools here
    command.extend(args)

    if not help_arg:
        env["RM_API"] = api_url()

    process = subprocess.Popen(
        command,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE)

    stdout, stderr = process.communicate()
    print(stdout.decode("utf-8"), end="")
    print(stderr.decode("utf-8"), end="", file=sys.stderr)

    return process.returncode


class CliError(Exception):
    pass


def main():
    args = sys.argv[2:]  # remove dcos-riak & riak
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
