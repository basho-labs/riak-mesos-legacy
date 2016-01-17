#!/bin/bash

. $HOME/erlang/R16B02-basho8/activate
[[ -s "$HOME/.gvm/scripts/gvm" ]] && source "$HOME/.gvm/scripts/gvm"
gvm use go1.5
export PATH=$PATH:$GOPATH/bin:$HOME/.gvm/gos/go1.4/bin

cd $GOPATH/src/github.com/basho-labs/riak-mesos
make
