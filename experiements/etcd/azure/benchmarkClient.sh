#!/bin/bash

#./blast.sh words.txt serverIpPort command id
# $1 textfile to put and get from the server
# $2 serverIpPort
# $3 The command to execute (etcd client binary)
# $4 unique Id this client marks files with
# $5 The benchmark to run: options [YCSB-A, YCSB-B, PUT-THEN-GET,
# PUT-AND-GET

# 

#latency is measured in ms
LATENCY=latency$4.dat
#bandwidth is measured in r/s
BANDWIDTH=bandwidth$4.txt
#duration of a clients life

BENCHMARK=$5
#BENCMARK="YCSB-A"
#BENCMARK="YCSB-B"
#BENCMARK="PUT-THEN-GET"
#BENCMARK="PUT-AND-GET"

i=0

echo "" > $LATENCY
echo "" > $BANDWIDTH


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
            ETCDCTL_API=3  $3 --endpoints=$2:2379 put $i "$word"
        else
            let "i=$RANDOM%${#HASPUT[@]}"
            echo "get key $i val ${HASPUT[$i]}"
            ETCDCTL_API=3 $3 --endpoints=$2:2379 get $i
        fi
        #measure latency
        bn=$(($(date +%s%N)/1000000))
        lat=0
        let lat=bn-na
        timestamp=`date +%s%N`
        timestamps=`date +%s`
        let "timestamp=timestamp-start"
        echo "$timestamp, $lat" >> $LATENCY
        let "secbuc=timestamps-starts"
        echo "$secbuc" >> $BANDWIDTH
    done
fi

if [ "$BENCHMARK" == "PUT-AND-GET" ]; then
    for word in $(<$1)
    do
        a=$(($(date +%s%N)/1000000))
        ETCDCTL_API=3  $# --endpoints=$2:2379 put $i "$word"
        b=$(($(date +%s%N)/1000000))
        lat=0
        let lat=b-a
        timestamp=`date +%s%N`
        let "timestamp=timestamp-start"
        echo "$timestamp, $lat" >> $LATENCY
        echo "$i, $timestamp" >> $BANDWIDTH

        a=$(($(date +%s%N)/1000000))
        ETCDCTL_API=3  $# --endpoints=$2:2379 get $i
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
fi

if [ "$BENCHMARK" == "PUT-THEN-GET" ]; then 
    echo "TODO: Implement put then get"
fi




