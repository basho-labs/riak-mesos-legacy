export DEBIAN_FRONTEND=noninteractive
export HOME=/home/vagrant

# Install mesos, marathon, zk, docker
apt-key adv --keyserver keyserver.ubuntu.com --recv E56151BF
DISTRO=$(lsb_release -is | tr '[:upper:]' '[:lower:]')
CODENAME=$(lsb_release -cs)
echo "deb http://repos.mesosphere.io/${DISTRO} ${CODENAME} main" |  sudo tee /etc/apt/sources.list.d/mesosphere.list
apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9
echo "deb https://get.docker.io/ubuntu docker main" | sudo tee /etc/apt/sources.list.d/docker.list
apt-get -y update > /dev/null
apt-get -y upgrade > /dev/null
apt-get -y install software-properties-common
add-apt-repository -y ppa:openjdk-r/ppa
apt-get -y update > /dev/null
apt-get -y install make git gcc g++ curl
apt-get -y install python-dev libcppunit-dev libunwind8-dev autoconf autotools-dev libltdl-dev libtool autopoint libcurl4-openssl-dev libsasl2-dev
apt-get -y install openjdk-8-jdk default-jre python-setuptools python-protobuf
update-java-alternatives -s /usr/lib/jvm/java-1.8.0-openjdk-amd64
apt-get -y install libprotobuf-dev protobuf-compiler
apt-get -y install mesos marathon
apt-get -y install lxc-docker

service zookeeper restart
service mesos-slave restart
service mesos-master restart
service marathon restart

# Install Golang
apt-get -y update
apt-get -y upgrade
apt-get -y install git bison mercurial autoconf
cd $HOME/ && bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
[[ -s "$HOME/.gvm/scripts/gvm" ]] && source "$HOME/.gvm/scripts/gvm"
gvm install go1.4
gvm use go1.4
gvm install go1.5
mkdir -p /vagrant/goroot

# DCOS Prereq
apt-get -y install python-pip
apt-get -y install zip
apt-get -y install s3cmd

echo '# Golang' >> $HOME/.bashrc
echo '[[ -s "$HOME/.gvm/scripts/gvm" ]] && source "$HOME/.gvm/scripts/gvm"' >> $HOME/.bashrc
echo 'gvm use go1.5' >> $HOME/.bashrc
echo 'export GOPATH=/vagrant/goroot' >> $HOME/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin:$HOME/.gvm/gos/go1.5/bin:$HOME/bin' >> $HOME/.bashrc

# Fix permissions
chown -R vagrant $HOME
chgrp -R vagrant $HOME
