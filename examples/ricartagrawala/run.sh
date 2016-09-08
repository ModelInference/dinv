#!/bin/bash


#!/bin/bash

HOSTS=5
SLEEPTIME=20

DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/ricartagrawala
#ricart-agrawala test cases
function shutdown {
    kill `ps | pgrep ricart | awk '{print $1}'` > /dev/null
}

function install {
    echo "installing dinv"
    cd ~/go/src/bitbucket.org/bestchai/dinv
    sudo -E go install 
}


function runTest {
    cd $testDir
    pids=()
    for (( i=0; i<$HOSTS; i++))
    do
        echo $i
        go run ricartagrawala.go -id=$i -hosts=$HOSTS -time=$SLEEPTIME &
    done

    sleep 15
    cat out.txt
    shutdown
}


function instrument {
    dinv -i -v -dir=$testDir/$1
}

function runLogMerger {
    cd $testDir
    dinv -l -plan=SCM *d.txt *g.txt
}
    

function cleanup {
    cd $testDir
    movelogs
    shutdown
}    

function movelogs {
    cd $testDir
    shopt -s nullglob
    set -- *[gd].txt
    if [ "$#" -gt 0 ]
    then
        name=`date "+%m-%d-%y-%s"`
        mkdir old/$name
        mv *[gdt].txt old/$name
        mv *.dtrace old/$name
        mv *.gz old/$name
    fi
}

#runDaikon first preforms work on the trace files, then prints out the invareints detected.
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


if [ "$1" == "-c" ];
then
    cleanup
    exit
fi
install
runTest
runLogMerger
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanup




