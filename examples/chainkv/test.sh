#!/bin/bash


DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/chainkv
BLIND=0

function runTest {
    cd $testdir/run
    pids=()
    let sudolast="$1 - 1"
    #start n-1 nodes
    for (( i=1; i<=$1; i++))
    do
        echo starting $i
        go test $testDir/run/run_test.go -id=$i -hosts=$1 -end=$BLIND &
        pids[$i]=$!
    done

    sleep 1
    echo starting client    
    go test $testDir/run/run_test.go -id=0 -hosts=$1 -end=$BLIND &

    for (( i=1; i<=$sudolast; i++))
    do
        wait ${pids[$i]}
    done

    kill ${pids[$1]}
    shutdown
}

function runLogMerger {
    cd $testDir/run
    dinv -v -l -shiviz *Encoded.txt *Log.txt

    #for file in ./*__*;do 
    #    echo $file
    #    rm $file
    #done
}

function runDaikon {
    cd $testDir/run
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
    rm output.txt
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
    cat output.txt
}

function blind {
    cd $testDir

    if [ -f blind ]; then
        echo useblind
        BLIND=`cat blind`
    else
        echo genblind
        echo $[ RANDOM % 2] > blind
        BLIND=`cat blind`
    fi
    echo $BLIND
}

function shutdown {
    kill `ps | pgrep client | awk '{print $1}'` > /dev/null
    kill `ps | pgrep chainkv | awk '{print $1}'` > /dev/null
}

function cleanup {
 cd $testDir/run
 rm *.txt
 rm *.dtrace
 rm *.gz
 rm *.log
 shutdown
}

function cleanupcontrol {
 cd $testDir/runa
 rm *.alog
 }


function cleanupall {
    cleanup
    cleanupcontrol
}


if [ "$1" == "-c" ];
then
    cleanupall
    exit
fi
if [ "$1" == "-t" ];
then
    runTest 5
    runLogMerger
    runDaikon
else
    runTest 5
    cleanup
fi
if [ "$1" == "-d" ];
then
    exit
fi


