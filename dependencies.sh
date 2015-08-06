#!/bin/bash

mkdir -p $GOPATH/src/bitbucket.org/bestchai
cd $GOPATH/src/bitbucket.org/bestchai
hg clone https://bitbucket.org/bestchai/dinv

go get github.com/godoctor/godoctor/analysis/cfg
go get github.com/arcaneiceman/GoVector/govec/vclock
go get github.com/willf/bitset
go get golang.org/x/tools/go/loader
go get golang.org/x/tools/go/types
go get gopkg.in/eapache/queue.v1

go install bitbucket.org/bestchai/dinv
