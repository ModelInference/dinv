#!/bin/bash

Hosts=5
BasePort=4000
testDir=$GOPATH/src/github.com/acarb95/DistributedAsserts/tests/diningPhilosophers

function runTestPrograms {
    cd $testDir
    pwd
    for (( i=0; i<Hosts; i++))
    do
        # echo $i
        let "hostPort=i + BasePort"
        let "neighbourPort= (i+1)%Hosts + BasePort"
        go run diningphilosopher.go -mP $hostPort -nP $neighbourPort &
    done
    sleep 60
    kill `ps | grep phil | awk '{print $1}'`
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
    cat 400*-Log.txt >> $file_name
}

function runDaikon {
    cd $testDir
    for directory in ./daikon*; do
        java daikon.Daikon $directory/*.dtrace
        mv *.gz $directory
        java daikon.PrintInvariants $directory/*.inv.gz > $directory/daikon_output.txt
    done
}

function cleanUp {
    cd $testDir
    kill `ps | grep dining | awk '{print $1}'`
    rm -rf results
}

#Start Here
if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi

runTestPrograms
# sleep 5
# runLogMerger
# runDaikon
shivizMerge
