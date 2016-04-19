#!/bin/bash

HOSTS=3
SLEEPTIME=3

DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/ricartagrawala
#ricart-agrawala test cases
function shutdown {
    kill `ps | pgrep ricart | awk '{print $1}'` > /dev/null
}

function setup {
    for (( i=0; i<$1; i++))

    do
        go test $1 -id=$i -hosts=$2 &
    done
}

function runTest {
    for (( i=0; i<$2; i++))
    do
        go test $1 -id=$i -hosts=$2 >> $ioutput.txt &
        #go test $1 -id=$i -hosts=$2 &
    done
}

function testWrapper {
    echo testing $1
    runTest $1 $2
    sleep $3
    mkdir $1-txt
    mv *.txt $1-txt
    shutdown
}


function runTests {
    cd "test"
    testWrapper "hoststartup_test.go" $HOSTS $SLEEPTIME
    testWrapper "onehostonecritical_test.go" $HOSTS $SLEEPTIME
    testWrapper "onehostmanycritical_test.go" $HOSTS $SLEEPTIME
    testWrapper "allhostsonecritical_test.go" $HOSTS 10
    testWrapper "allhostsmanycriticals_test.go" $HOSTS 15
    testWrapper "halfhostsonecritical_test.go" $HOSTS $SLEEPTIME
    testWrapper "halfhostsmanycriticals_test.go" $HOSTS $SLEEPTIME
}

function instrument {
    dinv -i -v -dir=$testDir/$1
}

function runDaikon {
    cd $testDir/test
     for directory in ./*-txt; do
         echo $directory
         cd $directory
         dinv -v -l *Encoded.txt *Log.txt
         mkdir dinv-output
         mv *.dtrace dinv-output
         dinv -v -l -plan=NONE *Encoded.txt *Log.txt
         mkdir daikon-output
         mv *.dtrace daikon-output
         cd ..
     done
 }
 
function cleanup {
    cd $testDir/test
    rm -r ./*-txt
    rm *.txt
}    

if [ "$1" == "-c" ];
then
    cleanup
    exit
fi
runTests
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
#cleanup


