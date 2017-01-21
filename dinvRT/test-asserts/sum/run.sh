#!/bin/bash

testDir=$GOPATH/src/bitbucket.org/bestchai/dinv/dinvRT/test-asserts/sum

function runTestPrograms {
    go run server/server.go &
    sleep 2
    go run client/client.go &
    sleep 10
    killall server
    mkdir results
    mv *.txt results
}

function runLogMerger {
    cd $testDir/results
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
    cat client-Log.txt >> $file_name
    cat server-Log.txt >> $file_name
}

function runDaikon {
    cd $testDir/results
    for directory in ./daikon*; do
        java daikon.Daikon $directory/*.dtrace
        mv *.gz $directory
        java daikon.PrintInvariants $directory/*.inv.gz > $directory/daikon_output.txt
    done
}

function cleanUp {
    cd $testDir
    rm -rf results
}

if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi

#runTestPrograms
#sleep 5
runLogMerger
runDaikon
shivizMerge
