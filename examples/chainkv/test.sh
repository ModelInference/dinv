#!/bin/bash


DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/chainkv

function runTest {
    cd $testdir/run
    pids=()
    for (( i=1; i<=$1; i++))
    do
        echo starting $i
        go test $testDir/run/run_test.go -id=$i -hosts=$1 &
        pids[$i]=$!
    done
    sleep 1
    echo starting client    
    go test $testDir/run/run_test.go -id=0 -hosts=$1 &

    sleep 5
    shutdown
    #for (( i=0; i<$2; i++))
    #do
    #    wait ${pids[$i]}
    #done
}

function shutdown {
    kill `ps | pgrep client | awk '{print $1}'` > /dev/null
    kill `ps | pgrep chainkv | awk '{print $1}'` > /dev/null
}

function cleanup {
 cd $testDir/run
 rm *.txt
}


if [ "$1" == "-c" ];
then
    cleanup
    exit
fi
runTest $1
if [ "$1" == "-d" ];
then
    exit
fi


