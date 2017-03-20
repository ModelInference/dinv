#!/bin/bash

i=0
for word in $(<$1)
do
    ETCDCTL_API=3 ../bin/etcdctl --endpoints=localhost:2379 put $i "$word"
    ETCDCTL_API=3 ../bin/etcdctl --endpoints=localhost:2379 get $i
    i=$((i+1))
done


i=0
for word in $(<$1)
do
    ETCDCTL_API=3 ../bin/etcdctl --endpoints=localhost:2379 get $i
    i=$((i+1))
done

