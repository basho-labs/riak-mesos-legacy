# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"

  # Riak Explorer
  config.vm.network "forwarded_port", guest: 9000, host: 9000

  # Riak
  config.vm.network "forwarded_port", guest: 8098, host: 8098
  config.vm.network "forwarded_port", guest: 8087, host: 8087

  # Marathon
  config.vm.network "forwarded_port", guest: 8080, host: 8080

  # Mesos Master
  config.vm.network "forwarded_port", guest: 5050, host: 5050

  # Mesos Slave
  config.vm.network "forwarded_port", guest: 5051, host: 5051

  # zookeeper
  config.vm.network "forwarded_port", guest: 2181, host:2181

  config.vm.network "private_network", ip: "33.33.33.2", auto_config: false

  config.vm.synced_folder "./../../../../", "/riak-mesos"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048"
  end

  config.vm.provision "shell", inline: <<-SHELL
    # Host communication
    HOSTMACHINE=`netstat -rn | grep "^0.0.0.0 " | cut -d " " -f10`
    echo "$HOSTMACHINE	33.33.33.1" >> /etc/hosts

    # Mesos
    apt-key adv --keyserver keyserver.ubuntu.com --recv E56151BF
    DISTRO=$(lsb_release -is | tr '[:upper:]' '[:lower:]')
    CODENAME=$(lsb_release -cs)
    echo "deb http://repos.mesosphere.io/${DISTRO} ${CODENAME} main" | \
      tee /etc/apt/sources.list.d/mesosphere.list
    apt-get -y update
    apt-get -y install mesos marathon

    echo "33.33.33.2" > /etc/mesos-master/hostname
    echo "33.33.33.2" > /etc/mesos-master/ip

    echo "33.33.33.2" > /etc/mesos-slave/hostname
    echo "33.33.33.2" > /etc/mesos-slave/ip

    service zookeeper restart
    service mesos-master restart
    service mesos-slave restart
    service marathon restart
    MASTER=$(mesos-resolve `cat /etc/mesos/zk` 2>/dev/null)
    mesos-execute --master=$MASTER --name="cluster-test" --command="sleep 5"

    # Go
    if [ ! -f go1.4.2.linux-amd64.tar.gz ]; then
        curl -O https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz
    fi
    tar -C /usr/local -xzf go1.4.2.linux-amd64.tar.gz
  SHELL
end
