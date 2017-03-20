#!/bin/bash

#Launching Script for Etcd cluster

PEERS=$1

CLUSTERSTRING=""

for i in $(seq 1 $PEERS)
do
    CLUSTERSTRING=$CLUSTERSTRING"infra"`expr $i - 1`"=http://127.0.0."$i":2380,"
done

echo $CLUSTERSTRING

fuser -k 2380/tcp
rm -r *[0-9].etcd
sudo -E go install ../

for i in $(seq 1 $PEERS)
do
infra="infra"`expr $i - 1`
etcd --name $infra --initial-advertise-peer-urls http://127.0.0.$i:2380 \
  --listen-peer-urls http://127.0.0.$i:2380 \
  --listen-client-urls http://127.0.0.$i:2379,http://127.0.0.$i:2379 \
  --advertise-client-urls http://127.0.0.$i:2379 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster $CLUSTERSTRING \
  --initial-cluster-state new &
done
