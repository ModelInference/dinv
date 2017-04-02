#!/bin/bash

#latency is measured in ms
LATENCY=latency.dat
#bandwidth is measured in r/s
BANDWIDTH=bandwidth.txt
#duration of a clients life

BENCMARK="YCSB-A"
#BENCMARK="YCSB-B"
#BENCMARK="PUT-THEN-GET"
#BENCMARK="PUT-AND-GET"

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
starts=`date +%s`
if [ "$BENCMARK" == "YCSB-A" ] || [ "$BENCHMARK" == "YCSB-B" ]; then
    if [ "$BENCMARK" == "YCSB-A" ]; then
        mod=2
    elif [ "$BENCHMARK" == "YSCB-B" ]; then
        mod=20
    fi

    HASPUT=()
    for word in $(<$1)
    do
        #put or get based on uniform modulo
        let "op=RANDOM%mod"
        echo "$op"
        an=$(($(date +%s%N)/1000000))
        if [ "$op" == "0" ];then
            i=${#HASPUT[@]}
            echo "put key: ${#HASPUT[@]} val: $word"
            HASPUT+=($word)
            ETCDCTL_API=3  $etcdctl --endpoints=localhost:2379 put $i "$word"
        else
            let "i=$RANDOM%${#HASPUT[@]}"
            echo "get key $i val ${HASPUT[$i]}"
            ETCDCTL_API=3 $etcdctl --endpoints=localhost:2379 get $i
        fi
        #measure latency
        bn=$(($(date +%s%N)/1000000))
        lat=0
        let lat=b-a
        timestamp=`date +%s%N`
        timestamps=`date +%s`
        let "timestamp=timestamp-start"
        echo "$timestamp, $lat" >> $LATENCY
        let "secbuc=timestamps-starts"
        echo "$secbuc" >> $BANDWIDTH
    done
fi


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

