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


function setup {
    for (( i=0; i<$1; i++))
    do
        go test $1 -id=$i -hosts=$2 &
    done
}

function runTest {
    pids=()
    for (( i=0; i<$2; i++))
    do
        go test $1 -id=$i -hosts=$2 -time=$3 >> passfail.stext &
        pids[$i]=$!
    done


    for (( i=0; i<$2; i++))
    do
        wait ${pids[$i]}
    done
}

function runOneMutant {
    pids=()
    go test $1 -id=0 -hosts=$2 -time=$3 >> passfail.stext &
    cp ../backup/ricartagrawala.go ../ricartagrawala.go
    for (( i=1; i<$2; i++))
    do
        go test $1 -id=$i -hosts=$2 -time=$3 >> passfail.stext &
        pids[$i]=$!
    done


    for (( i=0; i<$2; i++))
    do
        wait ${pids[$i]}
    done
}


function testWrapper {
    echo testing $1
    echo testing $1 >> passfail.stext
    runTest $1 $2 $3
    mkdir $1-txt
    mv *.txt $1-txt
    shutdown
}


function runTests {
    cd $testdir/test
    #testWrapper "hoststartup_test.go" $HOSTS $SLEEPTIME
    #testWrapper "onehostonecritical_test.go" $HOSTS $SLEEPTIME
    #testWrapper "onehostmanycritical_test.go" $HOSTS $SLEEPTIME
    #testWrapper "allhostsonecritical_test.go" $HOSTS $SLEEPTIME
    testWrapper "allhostsmanycriticals_test.go" $HOSTS $SLEEPTIME
    #testWrapper "halfhostsonecritical_test.go" $HOSTS $SLEEPTIME
    #testWrapper "halfhostsmanycriticals_test.go" $HOSTS $SLEEPTIME
}

function instrument {
    dinv -i -v -dir=$testDir/$1
}

function runLogMerger {
    cd $testDir/test
     for directory in ./*-txt; do
         echo $directory
         cd $directory
         #merging consistant cuts
         dinv -v -l *Encoded.txt *Log.txt
         mkdir dinv-output
         mv *.dtrace dinv-output
         #regualr daikon output
         dinv -l -plan=NONE *Encoded.txt *Log.txt 
         mkdir daikon-output
         mv *.dtrace daikon-output
         cd ..
     done
}

function sortOutput {
    cd $testDir/test
        let "i = 0"
        for directory in ./*-txt; do
            #sort dinv's output
            cd $directory/dinv-output
            for file in ./*; do
                #directory does not exist
                cleanName=`echo $file | sed 's/[:\/]//g'`
                if [ ! -d ../../dinv-$cleanName ]; then
                    mkdir ../../dinv-$cleanName
                fi

                mv $file ../../dinv-$cleanName/$i.dtrace
                let "i = i + 1"
            
            done

            cd ../..

            cd $directory/daikon-output
            for file in ./*; do
                #directory does not exist
                cleanName=`echo $file | sed 's/[:\/]//g'`
                if [ ! -d ../../daikon-$cleanName ]; then
                    mkdir ../../daikon-$cleanName
                fi

                mv $file ../../daikon-$cleanName/$i.dtrace
                let "i = i + 1"
            
            done
            #sort diakons output
            #for file in $directory/daikon-output/*; do
                #directory does not exist
            #    if [ ! -d daikon-$file ]; then
            #    fi

            #    mv $file daikon-$file/$directory_$file
            #done
        done
    }



function runDaikon {
    cd $testDir/test
    for directory in ./daikon*; do
        java daikon.Daikon $directory/*
        mv *.gz $directory
        gunzip $directory/*.gz
    done

    cd $testDir/test
    for directory in ./dinv*; do
        java daikon.Daikon $directory/*
        mv *.gz $directory
        gunzip $directory/*.gz
    done
}


 
function cleanup {
    cd $testDir/test
    movelogs
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
runTests
runLogMerger
#sortOutput
#runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
#cleanup


