#!/bin/bash

#latency is measured in ms
LATENCY=latency.dat
#bandwidth is measured in r/s
BANDWIDTH=bandwidth.txt
#duration of a clients life

RUNTIME=$2

echo "Running Client for $RUNTIME (s)"

i=0

echo "" > $LATENCY
echo "" > $BANDWIDTH

etcdctl=~/go/src/github.com/coreos/etcd/bin/etcdctl

self=$$
(
    sleep $RUNTIME;
    kill -9 $self;
) &

start=`date +%s%N`

for word in $(<$1)
do
    a=$(($(date +%s%N)/1000000))
    ETCDCTL_API=3  $etcdctl --endpoints=localhost:2379 put $i "$word"
    b=$(($(date +%s%N)/1000000))
    lat=0
    let lat=b-a
    timestamp=`date +%s%N`
    let "timestamp=timestamp-start"
    echo "$timestamp, $lat" >> $LATENCY
    echo "$i, $timestamp" >> $BANDWIDTH


    #kill the run after a given number of itterations
    #i=$((i+1))
    #echo $i
    #if [ "$i" -eq "100" ];then
    #    exit
    #fi

    echo "done"
done


#i=0
#for word in $(<$1)
#do
#    ETCDCTL_API=3 ../bin/etcdctl --endpoints=localhost:2379 get $i
#    i=$((i+1))
#done

