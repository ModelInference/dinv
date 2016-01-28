#!/bin/bash
# diningPhil/run.sh controls the exection of the dining philosophers
# example diningPhilosophers runs on an arbetrary number of hosts, the
# communication pattern follows Host_i-1 <--> Host_i <--> Host_i+1
# That is, every host has a neighbour, and only communicates with that
# neighbour

Hosts=5
BasePort=4000
DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/diningPhil
P1=phil

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function instrument {
    dinv -i -v -dir=$testDir/$1
}

function runTestPrograms {
    cd $testDir/$P1
    pwd
    for (( i=0; i<Hosts; i++))
    do
        let "hostPort=i + BasePort"
        let "neighbourPort= (i+1)%Hosts + BasePort"
        go run diningphilosophers.go -mP $hostPort -nP $neighbourPort &
    done
    sleep 60
    kill `ps | pgrep dining | awk '{print $1}'`
}

function runLogMerger {
    cd $testDir/diningPhil
    dinv -v -l -name="philosophers" -shiviz *Encoded.txt *Log.txt
}

function shivizMerge {
    cat $rm $DINV/TestPrograms/$TEST/$P1.log-Log.txt $DINV/TestPrograms/$TEST/$P2.log-Log.txt $DINV/TestPrograms/$TEST/$P3.log-Log.txt > $DINV/TestPrograms/expr/dinv_T3/shiviz.txt
    
}

function runDaikon {
    cd $testDir/diningPhil
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
    rm output.txt
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
    cat output.txt
}

function fixModDir {
    rm -r $testDir/$1
    mv $testDir/$1_orig $testDir/$1
}

function cleanUp {
    cd $testDir
    kill `ps | pgrep dining | awk '{print $1}'`
    fixModDir $P1
    rm ./*.dtrace
    rm ./*.gz
    rm ./*.txt
    
}

#Start Here
if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi
time installDinv
time instrument $P1
time runTestPrograms
time runLogMerger
time runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanUp
