#!/bin/bash
# sum/run.sh controls the execution of the sum client server program. client
# sends two random integers to server, and server responds with the sum.

# This script mannages the instrumentation and execution of these programs, as
# well as the merging of their generated logs, and execution of daikon on their
# generated trace files.

# The detected data invarients should include term1 + term2 = sum

#Options
#   -d : dirty run, all generated files are left after execution for
#   inspection
#   -c : cleanup, removes generated files created during the run

P1="server"
P2="client"
TEST=""
DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/sum


function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function instrument {
    dinv -i  -dir=$testDir/$1/lib
}


function fixModDir {
    if [ -d "$testDir/$1/"lib_orig ]; then
        rm -r $testDir/$1/lib
        mv $testDir/$1/lib_orig $testDir/$1/lib
    fi
}

function runTestPrograms {
    cd $testDir
    go run server/server.go &
    go run client/client.go &
    wait $!
    killall server
}


function runLogMerger {
 cd $testDir
 mv $1/*.txt ./
 mv $2/*.txt ./
 dinv  -logmerger -shiviz *Encoded.txt *Log.txt
}


function runDaikon {
    cd $testDir
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
    clear
    cat output.txt
}

function shivizMerge {
    cat $DINV/slog.log-Log.txt $DINV/testclient.log-Log.txt > ~/Research/expr/dinv_T2/shiviz.txt
}

function cleanUp {
    rmCreated
    kill `ps | pgrep Entry | awk '{print $1}'`
    fixModDir client
    fixModDir server
    cd $testDir
    rm *.dtrace
    rm *.inv.gz
    rm *.txt
}


function rmCreated {
 cd $testDir
 rm *.txt
}

if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi
installDinv
instrument client
instrument server
runTestPrograms
runLogMerger client server
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanUp
