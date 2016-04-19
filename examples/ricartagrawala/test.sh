#!/bin/bash

HOSTS=3
SLEEPTIME=7

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
        #go test $1 -id=$i -hosts=$2 >> output.txt &
        go test $1 -id=$i -hosts=$2 &
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

function runDaikon {
     for directory in *-txt; do
         echo $directory
         cd $directory
         dinv -v -l *Encoded.txt *Log.txt
         mkdir dinv
         mv *.dtrace dinv
         dinv -v -l -plan=NONE *Encoded.txt *Log.txt
         mkdir daikon
         mv *.dtrace daikon
     done
 }
 

runTests
runDaikon

