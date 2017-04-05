#!/bin/bash
# measure sets up a bunch of measuring clients
# $1 Text file to put and get on a server
# $2 [server ip:port] running etcd
# $3 the time the clients should run for
# $4 number of clients
# $5 etcd client exacutable location
# $6 client script for measuring or just execution on the server
# $7 The benchmark to run: options [YCSB-A, YCSB-B, PUT-THEN-GET,
# PUT-AND-GET]

#
i=0
CLIENTS=$4

for (( i=0; i<CLIENTS; i++ ))
do
    $6 $1 $2 $5 $i $7 &
done

echo "RUNTIME $3"
sleep $3;
killall blast.sh
kill -9 $self;


