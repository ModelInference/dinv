#!/bin/bash

#Master is a control script for running an etcd cluster, and client locally.
#Master launches a client using the "clientMeasure.sh" or "clientblast.sh" scripts.
#The cluster is launched via modcluster which takes in "clustersize" as a parameter.

#input files
#dog.in the quick
#kahn.in kublakahn
#dec.in declaration
#in.in test*30000
#/usr/share/dict/words words
killall etcd
killall clientMeasure.sh

clustersize=3
clients=50
clientRuntime=10
etcddir=~/go/src/github.com/coreos/etcd

./clean.sh
sudo -E go install $etcddir

if [ $1 == -b ]; then
    exit
fi

rm *.time
rm *.txt
./modcluster.sh $clustersize &
sleep 3
#./clientblast.sh /usr/share/dict/words &
for (( i=0; i < $clients; i++)); do
    ./clientMeasure.sh /usr/share/dict/words $clientRuntime &
done
sleep $clientRuntime
killall etcd
killall clientMeasure.share

./statplot.sh
#cat *.time | grep time:
#ls -lrt *.txt | nawk '{print $5}' | awk '{total = total + $1}END{print total}'
#time dinv -l -plan=SCM -json -name=fruits -shiviz *d.txt *g.txt
#./daikon.sh
