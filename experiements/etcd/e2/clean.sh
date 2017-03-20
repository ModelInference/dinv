#!/bin/bash

testDir=~/go/src/github.com/coreos/etcd/dinv




function movelogs {
    cd $testDir
    shopt -s nullglob
    set -- *[gd].txt
    if [ "$#" -gt 0 ]
    then
        name=`date "+%m-%d-%y-%s"`
        mkdir old/$name
        mv *[gd].txt old/$name
        mv *.dtrace old/$name
        mv *.gz old/$name
    fi
}

movelogs
