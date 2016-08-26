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
P1=diningphilosopher.go
Original=original

function installDinv {
    echo "Install dinv"
    cd $DINV
    sudo -E go install
}

function instrument {
    cd $testDir
    mkdir $Original
    cp $P1 $Original/

    dinv -i  -file=$P1
    GoVector  -file=$P1
}

function runTestPrograms {
    cd $testDir
    pwd
    for (( i=0; i<Hosts; i++))
    do
        let "hostPort=i + BasePort"
        let "neighbourPort= (i+1)%Hosts + BasePort"
        go run diningphilosopher.go -mP $hostPort -nP $neighbourPort &
    done
    sleep 8
    kill `ps | pgrep dining | awk '{print $1}'`
}

function runLogMerger {
    cd $testDir/diningPhil
    dinv -v -l -plan="SCM" -shiviz *Encoded.txt *Log.txt
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

function fixModDir {
    cd $testDir
    if [ -d $Original ]; then
            rm $P1
            mv $Original/* ./
            rmdir $Original
    fi
}

function cleanUp {
    cd $testDir
    kill `ps | pgrep dining | awk '{print $1}'`
    fixModDir $P1
    rm ./*.dtrace
    rm ./*.gz
    rm ./*.txt
    rm L*
    
}

#Start Here
if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi
installDinv
instrument $P1
runTestPrograms
runLogMerger
time runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanUp
