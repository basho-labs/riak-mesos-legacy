# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"

  # Marathon
  config.vm.network "forwarded_port", guest: 8080, host: 8080

  # Mesos Master
  config.vm.network "forwarded_port", guest: 5050, host: 5050

  # Mesos Slave
  config.vm.network "forwarded_port", guest: 5051, host: 5051

  # zookeeper
  config.vm.network "forwarded_port", guest: 2181, host:2181

  config.vm.network "private_network", ip: "33.33.33.2"

  config.vm.hostname = "ubuntu.local"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048"
  end

  $script = <<-SCRIPT
    # Host communication
    HOSTMACHINE=`netstat -rn | grep "^0.0.0.0 " | cut -d " " -f10`
    echo "$HOSTMACHINE	33.33.33.1" >> /etc/hosts
    DISTRO=$(lsb_release -is | tr '[:upper:]' '[:lower:]')
    CODENAME=$(lsb_release -cs)
    echo "deb http://repos.mesosphere.io/${DISTRO} ${CODENAME} main" | \
    tee /etc/apt/sources.list.d/mesosphere.list
    # Mesos
    apt-key adv --keyserver keyserver.ubuntu.com --recv E56151BF
    apt-get -y update
    apt-get -y install mesos marathon
    chown -R zookeeper /var/lib/zookeeper
    chgrp -R zookeeper /var/lib/zookeeper
    echo "ubuntu.local" > /etc/mesos-master/hostname
    echo "33.33.33.2" > /etc/mesos-master/ip
    echo "ubuntu.local" > /etc/mesos-slave/hostname
    echo "33.33.33.2" > /etc/mesos-slave/ip
    service zookeeper restart
    service mesos-master restart
    service mesos-slave restart
    service marathon restart
    # MASTER=$(mesos-resolve `cat /etc/mesos/zk` 2>/dev/null)
    # mesos-execute --master=$MASTER --name="cluster-test" --command="sleep 5"
  SCRIPT

  config.vm.provision "shell", inline: $script
end
