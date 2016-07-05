#!/bin/bash
hg revert ClientServer.go
dinv -i -v -file=ClientServer.go
gofmt -w=true ClientServer.go
echo "Insturmented Run"
runlim go run ClientServer.go > /dev/null

sleep 1
dinv -i -c -v -file=ClientServer.go
gofmt -w=true ClientServer.go
echo "Control Run"
runlim go run ClientServer.go > /dev/null
