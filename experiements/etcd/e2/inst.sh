#!/bin/bash

cd ~/go/src/github.com/coreos/etcd
rm -r raft
rm -r raft_orig
rm -r rafthttp
rm -r rafthttp_orig
git checkout raft
git checkout rafthttp

