#!/bin/bash
hg revert ClientServer.go
sudo -E go install ../../
dinv -i  -file=ClientServer.go
gofmt -w=true ClientServer.go
sleep 1
dinv -i -c  -file=ClientServer.go
