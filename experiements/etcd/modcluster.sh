#!/bin/bash

#Launching Script for Etcd cluster
#Mod cluster takes in as an argument the number of nodes a cluster should run

PEERS=$1

CLUSTERSTRING=""
DINV_ASSERT_PEERS=""

#TERMINAL=gnome-terminal -x
TERMINAL=""

etcddir=~/go/src/github.com/coreos/etcd

#Bulild port and ip number for etcd and dinv assertions
for i in $(seq 1 $PEERS)
do
    #port and ip for etcd
    CLUSTERSTRING=$CLUSTERSTRING"infra"`expr $i - 1`"=http://127.0.0."$i":2380,"
    #port ip for dinv assert default 12000
    DINV_ASSERT_PEERS=$DINV_ASSERT_PEERS"127.0.0."$i":12000,"

done

echo $CLUSTERSTRING

#kill any old clusters
fuser -k 2380/tcp
#remove old databases
rm -r *[0-9].etcd
#install etcd
sudo -E go install ../

#hard coded enviornment variables for testing dinv assertions
#they type of assert to be made
ASSERTTYPE="STRONGLEADER"
#if true only leader asserts
LEADER="false"
#assert with probability 1/SAMPLE
SAMPLE="0"
#true if bugs should be run
DINVBUG="false"
#export assert macros
export LEADER
export ASSERTTYPE
export SAMPLE
export DINVBUG


#itterativly launch the cluster
for i in $(seq 1 $PEERS)
do
    infra="infra"`expr $i - 1`
    #Setup assert names, each node is given an ip port 127.0.0.(id):12000
    DINV_HOSTNAME="NODE"$i
    DINV_ASSERT_LISTEN="127.0.0."$i":12000"
    #export each of the names before launching a node
    export DINV_HOSTNAME
    export DINV_ASSERT_PEERS
    export DINV_ASSERT_LISTEN
    #launch the nodes
    $TERMINAL etcd --name $infra --initial-advertise-peer-urls http://127.0.0.$i:2380 \
      --listen-peer-urls http://127.0.0.$i:2380 \
      --listen-client-urls http://127.0.0.$i:2379,http://127.0.0.$i:2379 \
      --advertise-client-urls http://127.0.0.$i:2379 \
      --initial-cluster-token etcd-cluster-1 \
      --initial-cluster $CLUSTERSTRING \
      --initial-cluster-state new &
done
