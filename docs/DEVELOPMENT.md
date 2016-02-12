# Development Guide

### Vagrant Dev Environment

For a Mesos / Vagrant / Erlang Vagrant environment, follow directions here: [mesos-erlang-vagrant](https://github.com/drewkerrigan/mesos-erlang-vagrant)

### Go installation

```
sudo apt-get -y update
sudo apt-get -y upgrade
sudo apt-get -y install git bison mercurial autoconf
cd $HOME/ && bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
[[ -s "$HOME/.gvm/scripts/gvm" ]] && source "$HOME/.gvm/scripts/gvm"
gvm install go1.4
gvm use go1.4
mkdir -p ~/go
export GOPATH=~/go
gvm install go1.5
gvm use go1.5
export GOPATH=~/go
export PATH=$PATH:$GOPATH/bin:$HOME/.gvm/gos/go1.5/bin:$HOME/bin
### .bashrc changes
echo '# Golang' >> $HOME/.bashrc
echo '[[ -s "$HOME/.gvm/scripts/gvm" ]] && source "$HOME/.gvm/scripts/gvm"' >> $HOME/.bashrc
echo 'gvm use go1.5' >> $HOME/.bashrc
echo 'export GOPATH=~/go' >> $HOME/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin:$HOME/.gvm/gos/go1.5/bin:$HOME/bin' >> $HOME/.bashrc
```

Bring up the build environment with a running Mesos and ssh in

### Set up Dependencies

```
./setup-env.sh
```

### Build the Framework

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos && make
```

or for a faster build with lower RAM requirements:

```
cd $GOPATH/src/github.com/basho-labs/riak-mesos && TAGS=dev make
```
