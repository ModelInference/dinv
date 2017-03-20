#!/bin/bash

HOSTS=5
SLEEPTIME=20

testDir=$GOPATH/src/github.com/acarb95/DistributedAsserts/tests/ricartagrawala

function shutdown {
    kill `ps | grep ricart | awk '{print $1}'` > /dev/null
}

function runTest {
    pids=()
    for (( i=0; i<$HOSTS; i++))
    do
        # echo $i
        go run ricartagrawala.go -id=$i -hosts=$HOSTS -time=$SLEEPTIME &
    done

    sleep 60
    shutdown
    mkdir results
    mv *.txt results
}

function runLogMerger {
    cd $testDir/results
    #regular daikon output
    dinv -l -plan=SCM *Encoded.txt *log-Log.txt 
    mkdir daikon-output
    mv *.trace daikon-output
    for trace_file in ./daikon-output/*; do
        mv "$trace_file" "./daikon-output/$(basename "$trace_file" .trace).dtrace"
    done
}

function shivizMerge {
    cd $testDir/results
    file_name=./AssertionShiViz.log
    echo "(?<host>\S*) (?<clock>{.*})\\n(?<event>.*)" > $file_name
    echo "" >> $file_name
    cat node*-Log.txt >> $file_name
}
    

function cleanup {
    cd $testDir
    movelogs
}    

function movelogs {
    rm -rf results
}

#runDaikon first preforms work on the trace files, then prints out the invariants detected.
function runDaikon {
    cd $testDir/results
    for directory in ./daikon*; do
        java daikon.Daikon $directory/*.dtrace
        mv *.gz $directory
        java daikon.PrintInvariants $directory/*.inv.gz > $directory/daikon_output.txt
    done
}


if [ "$1" == "-c" ];
then
    cleanup
    exit
fi
runTest
sleep 5
runLogMerger
runDaikon
shivizMerge
