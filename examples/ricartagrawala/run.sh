#!/bin/bash


#!/bin/bash

HOSTS=3
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
    cd $testDir/test
    pids=()
    for (( i=0; i<$HOSTS; i++))
    do
        go test "allhostsmanycriticals_test.go" -id=$i -hosts=$HOSTS -time=$SLEEPTIME &
        pids[$i]=$!
    done

    for (( i=0; i<$2; i++))
    do
        wait ${pids[$i]}
    done
}


function instrument {
    dinv -i -v -dir=$testDir/$1
}

function runLogMerger {
    cd $testDir/test
    dinv -l *d.txt *g.txt
}
    

function cleanup {
    cd $testDir/test
    movelogs
    rm -r ./*-txt
    rm -r dinv*
    rm -r daikon*
    rm *.gz
    rm *.txt
    rm *.stext
}    

function movelogs {
    cd $testDir/test
    shopt -s nullglob
    set -- *[gd].txt
    if [ "$#" -gt 0 ]
    then
        name=`date "+%m-%d-%y-%s"`
        mkdir old/$name
        mv *[gd].txt ../old/$name
        mv *.dtrace ../old/$name
        mv *.gz ../old/$name
    fi
}


if [ "$1" == "-c" ];
then
    cleanup
    exit
fi
install
runTest
runLogMerger
#sortOutput
#runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
#cleanup




