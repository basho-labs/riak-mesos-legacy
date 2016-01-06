#!/usr/bin/env bash

# Capture the output of all of the commands
OUTPUT_LOG=/tmp/riaks2_deploy.log
OUTPUT_PIPE=/run/user/1000/riaks2_deploy-output.pipe

if [ ! -e ${OUTPUT_PIPE} ]; then
    mkfifo ${OUTPUT_PIPE}
fi

if [ -e ${OUTPUT_LOG} ]; then
    rm ${OUTPUT_LOG}
fi

exec 3>&1 4>&2
tee ${OUTPUT_LOG} < ${OUTPUT_PIPE} >&3 &
tpid=$!
exec > ${OUTPUT_PIPE} 2>&1

CWD=${PWD}
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

## Configuration items
# Number of Riak Nodes in cluster
readonly NUM_NODES=3
readonly DIRECTOR_IP=192.168.0.7
readonly USEREMAIL="admin@email.com"
readonly USERNAME="admin"

echo ""
echo "### Starting the Framework"
echo ""

dcos package install --yes riak --options=dcos-riak.json

echo ""
echo ""
echo "Waiting for the health check for riak to turn green in http://leader.mesos:8080/ui/#/apps"
echo ""

until [[ "1" -eq "$(curl --silent http://leader.mesos:8080/v2/apps/riak | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasksHealthy"]')" ]]; do printf "."; sleep 1s; done

echo ""
echo "Riak Mesos Framework is up"
echo ""

echo ""
echo "### Creating a cluster of 3 nodes"
echo ""

cd "${DIR}" || exit
FRAMEWORKHOST=$(curl --silent http://leader.mesos:8080/v2/apps/riak | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["host"]')
FRAMEWORKPORT=$(curl --silent http://leader.mesos:8080/v2/apps/riak | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][0]')

curl -XPUT ${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster

sleep 5s

curl -v -XPOST ${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/config --data-binary @etc/riak.conf
curl -v -XPOST ${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/advancedConfig --data-binary @etc/advanced.config

sleep 5s

for n in $(seq 0 $((NUM_NODES - 1))); do
    curl -XPOST ${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes
done

echo ""
echo ""
echo "Waiting for the Riak Mesos Framework to return 3 nodes"
echo ""

until [[ "${NUM_NODES}" -eq "$(curl --silent http://${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes | python -c 'import sys, json; print len(json.load(sys.stdin))')" ]]; do printf "."; sleep 1s; done

echo ""
echo ""
for n in $(seq 0 $((NUM_NODES - 1))); do
    echo "Waiting for the Riak Node #$((n + 1)) to return OK from /ping"
    echo ""

    RIAKNODE[n]=$(curl --silent http://${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes | python -c "import sys, json; print json.load(sys.stdin).keys()[$n]")
    RIAKHTTPHOST[n]=$(curl --silent http://${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes | python -c "import sys, json; print json.load(sys.stdin)['${RIAKNODE[n]}']['TaskData']['Host']")
    RIAKHTTPPORT[n]=$(curl --silent http://${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes | python -c "import sys, json; print json.load(sys.stdin)['${RIAKNODE[n]}']['TaskData']['HTTPPort']")

    until [[ "2" -eq "$(curl --silent http://${RIAKHTTPHOST[n]}:${RIAKHTTPPORT[n]}/ping | python -c 'import sys; print len(sys.stdin.read())')" ]]; do
        RIAKHTTPHOST[n]=$(curl --silent http://${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes | python -c "import sys, json; print json.load(sys.stdin)['${RIAKNODE[n]}']['TaskData']['Host']")
        RIAKHTTPPORT[n]=$(curl --silent http://${FRAMEWORKHOST}:${FRAMEWORKPORT}/api/v1/clusters/mycluster/nodes | python -c "import sys, json; print json.load(sys.stdin)['${RIAKNODE[n]}']['TaskData']['HTTPPort']")
        printf "."
        sleep 5s
    done
    echo ""
done

echo ""
echo "### Deploying the Director"
echo ""

dcos riak proxy install --os centos --cluster mycluster
dcos riak proxy endpoints --public-dns "${DIRECTOR_IP}"

echo ""
echo ""
echo "Waiting for the health check for riak-director to turn green in http://leader.mesos:8080/ui/#/apps"
echo ""

until [[ "1" -eq "$(curl --silent http://leader.mesos:8080/v2/apps/riak-director | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasksHealthy"]')" ]]; do printf "."; sleep 1s; done

echo ""
echo "### Finding available endpoints"
echo ""
HOST=$(curl --silent http://leader.mesos:8080/v2/apps/riak-director | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["host"]')
DIRPORT=$(curl --silent http://leader.mesos:8080/v2/apps/riak-director | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][2]')
HTTPPORT=$(curl --silent http://leader.mesos:8080/v2/apps/riak-director | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][0]')
PBPORT=$(curl --silent http://leader.mesos:8080/v2/apps/riak-director | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][1]')

for n in $(seq 0 $((NUM_NODES - 1))); do
    RIAKHOST[n]=$(curl --silent ${HOST}:${DIRPORT}/nodes | python -c 'import sys, json; print json.load(sys.stdin)["nodes"]['${n}']["http"]["host"]')
    RIAKHTTPPORT[n]=$(curl --silent ${HOST}:${DIRPORT}/nodes | python -c 'import sys, json; print json.load(sys.stdin)["nodes"]['${n}']["http"]["port"]')
    RIAKPBPORT[n]=$(curl --silent ${HOST}:${DIRPORT}/nodes | python -c 'import sys, json; print json.load(sys.stdin)["nodes"]['${n}']["protobuf"]["port"]')
done

echo "##### Riak HTTP Nodes:"
echo ""
for n in $(seq 0 $((NUM_NODES - 1))); do
    echo "* [${RIAKHOST[n]}:${RIAKHTTPPORT[n]}](http://${RIAKHOST[n]}:${RIAKHTTPPORT[n]})"
done
echo ""

echo "##### Riak PB Nodes:"
echo ""
for n in $(seq 0 $((NUM_NODES - 1))); do
    echo "* ${RIAKHOST[n]}:${RIAKPBPORT[n]}"
done
echo ""

echo "##### Other Endpoints:"
echo ""
echo "* Director Node List: [${HOST}:${DIRPORT}/nodes](http://${HOST}:${DIRPORT}/nodes)"
echo "* Director Status:    [${HOST}:${DIRPORT}/status](http://${HOST}:${DIRPORT}/status)"
echo "* Riak HTTP Proxy:    [${HOST}:${HTTPPORT}](http://${HOST}:${HTTPPORT})"
echo "* Riak PB Proxy:      ${HOST}:${PBPORT}"
echo "* Riak Bucket Types:  [${HOST}:${HTTPPORT}/admin/explore/clusters/default/bucket_types](http://${HOST}:${HTTPPORT}/admin/explore/clusters/default/bucket_types)"
echo "* Riak Member Status: [${HOST}:${HTTPPORT}/admin/control/clusters/default/status](http://${HOST}:${HTTPPORT}/admin/control/clusters/default/status)"
echo "* Riak Ring Ready:    [${HOST}:${HTTPPORT}/admin/control/clusters/default/ringready](http://${HOST}:${HTTPPORT}/admin/control/clusters/default/ringready)"

# Make the JSON file from the template with the host and port from RIAKHOST[0]
perl -pe "s/{{.RIAKHOSTPORT}}/${RIAKHOST[0]}:${RIAKPBPORT[0]}/" riak-s2-init.json.template > riak-s2-init.json
# Deploy riak-s2-init to RIAKHOST[0]
echo ""
echo "Deploying initial Riak S2 deployment..."
curl -v -XPOST http://leader.mesos:8080/v2/apps -d @./riak-s2-init.json -H "Content-Type: application/json"

echo ""
echo ""
echo "Waiting for the health check for the initial Riak S2 deployment to turn green in http://leader.mesos:8080/ui/#/apps"
echo ""

until [[ "1" -eq "$(curl --silent http://leader.mesos:8080/v2/apps/riak-s2-init | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasksHealthy"]')" ]]; do printf "."; sleep 1s; done

# Query Mesos to get HOST and PORT0 for riak-s2-init
INITHOST=$(curl --silent http://leader.mesos:8080/v2/apps/riak-s2-init | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["host"]')
INITPORT=$(curl --silent http://leader.mesos:8080/v2/apps/riak-s2-init | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][0]')

# Create the admin user
RESULTS=$(curl -H 'Content-Type: application/json' \
               -XPOST http://${INITHOST}:${INITPORT}/riak-cs/user \
               --data '{"email":"'${USEREMAIL}'", "name": "'${USERNAME}'"}')
ADMIN_KEY=$(/usr/bin/python -c "import json; print json.loads('$RESULTS')[u'key_id']")
ADMIN_SECRET=$(/usr/bin/python -c "import json; print json.loads('$RESULTS')[u'key_secret']")
echo ""
echo "## Admin user created:"
echo "   * Admin key: ${ADMIN_KEY}"
echo "   * Admin secret: ${ADMIN_SECRET}"
echo ""

# Destroy the riak-s2-init app instance
curl -v -XDELETE http://leader.mesos:8080/v2/apps/riak-s2-init -H "Content-Type: application/json"

# Generate Stanchion JSON file
perl -p  -e "s/{{.RIAKHOSTPORT}}/${RIAKHOST[0]}:${RIAKPBPORT[0]}/" stanchion.json.template > stanchion.json
perl -pi -e "s/{{.ADMIN_KEY}}/${ADMIN_KEY}/" stanchion.json
perl -pi -e "s/{{.ADMIN_SECRET}}/${ADMIN_SECRET}/" stanchion.json
# Deploy Stanchion App
curl -v -XPOST http://leader.mesos:8080/v2/apps -d @./stanchion.json -H "Content-Type: application/json"
sleep 5s
# Query Mesos to get PORT0 for stanchion
STANCHIONHOST=$(curl --silent http://leader.mesos:8080/v2/apps/stanchion | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["host"]')
STANCHIONPORT=$(curl --silent http://leader.mesos:8080/v2/apps/stanchion | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][0]')

for n in $(seq 0 $((NUM_NODES - 1))); do
    node_num=$((n + 1))
    json_file="./riak-s2-${node_num}.json"
    # Generate JSON file from template
    perl -p  -e "s/{{.S2NODENUM}}/${node_num}/" riak-s2.json.template > "${json_file}"
    perl -pi -e "s/{{.RIAKHOSTPORT}}/${RIAKHOST[n]}:${RIAKPBPORT[n]}/" "${json_file}"
    perl -pi -e "s/{{.STANCHIONHOSTPORT}}/${STANCHIONHOST}:${STANCHIONPORT}/" "${json_file}"
    perl -pi -e "s/{{.ADMIN_KEY}}/${ADMIN_KEY}/" "${json_file}"
    perl -pi -e "s/{{.ADMIN_SECRET}}/${ADMIN_SECRET}/" "${json_file}"
    perl -pi -e "s/{{.RIAKHOSTNAME}}/${RIAKHOST[n]}/" "${json_file}"
    # Deploy riak-s2
    echo ""
    echo "Deploying Riak S2 node ${node_num} to ${RIAKHOST[0]}"
    curl -v -XPOST http://leader.mesos:8080/v2/apps -d @"${json_file}" -H "Content-Type: application/json"

    echo ""
    echo ""
    echo "Waiting for the health check for the Riak S2 Node ${node_num} deployment to turn green in http://leader.mesos:8080/ui/#/apps"
    echo ""

    until [[ "1" -eq "$(curl --silent http://leader.mesos:8080/v2/apps/riak-s2-"${node_num}" | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasksHealthy"]')" ]]; do printf "."; sleep 1s; done
    echo ""

    # Query Mesos to get host and port for Riak S2
    RIAKS2HOST[n]=$(curl --silent http://leader.mesos:8080/v2/apps/riak-s2-"${node_num}" | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["host"]')
    RIAKS2PORT[n]=$(curl --silent http://leader.mesos:8080/v2/apps/riak-s2-"${node_num}" | python -c 'import sys, json; print json.load(sys.stdin)["app"]["tasks"][0]["ports"][0]')
done

echo "##### Riak S2 Nodes:"
echo ""
for n in $(seq 0 $((NUM_NODES - 1))); do
    echo "* riak-s2-$((n + 1)).marathon.mesos - ${RIAKS2HOST[n]}:${RIAKS2PORT[n]}"
done
echo ""


cd "${CWD}" || exit

exec 1>&3 3>&- 2>&4 4>&-
wait ${tpid}

rm ${OUTPUT_PIPE}
