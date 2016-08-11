#!/bin/bash
#hg revert ClientServer.go

sudo -E go install ../../
#sudo -E go install ../../../../../github.com/arcaneiceman/GoVector


#dinv -i -c -file=ClientServer.go
#gofmt -w=true ClientServer.go

#echo "Insturmentation Cost"
#hg revert ClientServer.go

echo "Control Run"
runlim go run ClientServer.go > /dev/null

#echo "Insturmentation Cost"
#hg revert ClientServer.go

#echo "dump insturmentation"
#runlim dinv -i -file=ClientServer.go

#echo "Vector Clock Instertion"
#runlim GoVector ClientServer.go

#echo "Insturmented Run"
#runlim go run ClientServer.go > /dev/null



