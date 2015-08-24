#!/bin/bash

peers=$1
time=$2

DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
RAFT=$GOPATH/src/github.com/hashicorp/raft
testDir=$DINV/examples/raft


function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function instrument {
    dinv -i -dir=$RAFT
}

function fixModDir {
    echo $RAFT
    echo "$RAFT"_orig
    if [ -d "$RAFT"_orig ]; then
        rm -r $RAFT
        mv "$RAFT"_orig $RAFT
    fi
}

function runTestPrograms {
    cd $testDir
    rm peers/peers.json

    peerList="["
    for (( i=0; i<peers; i++))
    do
        peerList="$peerList \"127.0.0.1:808$i\""
        plus=$(( $i + 1))
        if [ $plus -lt $peers ]; then
            peerList="$peerList,"
        fi
    done
    peerList="$peerList]"
    echo $peerList > peers/peers.json


    for (( i=0; i<peers; i++))
    do
        rm data/raft808$i.db
        touch data/raft808$i.db
    done

    for (( i=0; i<peers; i++))
    do
        go run main.go 808$i &
    done
    sleep $time

    kill -9 `ps -f| grep main | grep -v grep | awk '{print $2}'`

}

function runLogMerger {
    cd $testDir
    dinv -l -v -name="fruits" -shiviz *Encoded.txt *Log.txt
}

function runDaikon {
    cd $testDir
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
    rm output.txt
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
    cat output.txt
}

function cleanUp {
    fixModDir
    rm ./*.txt
    rm ./*.dtrace
    rm ./*.gz
    rm -r snapshots
    cd $testDir
    for (( i=0; i<=peers; i++))
    do
        rm data/raft808$i.db
    done
}

if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi
installDinv
instrument
runTestPrograms
runLogMerger
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
