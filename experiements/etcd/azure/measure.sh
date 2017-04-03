#!/bin/bash
# measure sets up a bunch of measuring clients
# measure.sh words.txt [server ip:port] runtime clients etcdcdlLoc
# (newline) client
#ex ./measure.sh /

#command line arguments
# $1: 
#
#
#
#
i=0
CLIENTS=$4

for (( i=0; i<CLIENTS; i++ ))
do
    $6 $1 $2 $5 $i &
done

echo "RUNTIME $3"
sleep $3;
killall blast.sh
kill -9 $self;


