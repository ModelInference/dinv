#!/bin/bash

#Launching Script for Etcd cluster

LOCAL=127.0.0.1
L1=127.0.0.2
L2=127.0.0.3
PEER1=1000
PEER2=2000
PEER3=3000

fuser -k 2380/tcp
rm -r *[0-9].etcd
sudo -E go install ../

etcd --name infra0 --initial-advertise-peer-urls http://$LOCAL:2380 \
  --listen-peer-urls http://$LOCAL:2380 \
  --listen-client-urls http://$LOCAL:2379,http://$LOCAL:2379 \
  --advertise-client-urls http://$LOCAL:2379 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster infra0=http://$LOCAL:2380,infra1=http://$L1:2380,infra2=http://$L2:2380 \
  --initial-cluster-state new &

etcd --name infra1 --initial-advertise-peer-urls http://$L1:2380 \
  --listen-peer-urls http://$L1:2380 \
  --listen-client-urls http://$L1:2379,http://$L1:2379 \
  --advertise-client-urls http://$L1:2379 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster infra0=http://$LOCAL:2380,infra1=http://$L1:2380,infra2=http://$L2:2380 \
  --initial-cluster-state new &

etcd --name infra2 --initial-advertise-peer-urls http://$L2:2380 \
  --listen-peer-urls http://$L2:2380 \
  --listen-client-urls http://$L2:2379,http://$L2:2379 \
  --advertise-client-urls http://$L2:2379 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster infra0=http://$LOCAL:2380,infra1=http://$L1:2380,infra2=http://$L2:2380 \
  --initial-cluster-state new

