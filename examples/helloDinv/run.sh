#!/bin/bash
#run.sh installs DInv and completes a full execution of the example program helloDinv
#This script oversees the execution of the two primary DInv features, the instrumenter and the logmerger
#Two test files, server.go and client.go are instrumented, the result being two modified files mod_server.go and mod_client.go
#The orignal files are moved into temporary directories, and the modified files are executed
#Post execution 6 log files will be generated, A readable, and encoded log of dump points for both client and server
#and a shiviz readable log for both client and server
#The logmerger is run on the logs, the result being a single daikon trace file
#lastly the trace file is given an input to daikon
#The detected invarients are the messages sent between the client and server.

#Default behaviour : Execute, and cleanup

#Options 
#   -d : dirty run, all generated files are left after execution for
#   inspection
#   -c : cleanup, removes generated files created during the run


testDir=$DINV/examples/helloDinv

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function instrument {
    dinv -i $testDir/$1
}

function fixModDir {
    rm -r $testDir/$1
    mv $testDir/$1_orig $testDir/$1
}

function runTestProgram {
    cd $testDir/$1
    go run $1.go &
    sleep 1
}

function cleanup {
    rmCreated client
    rmCreated server
    kill `ps | pgrep server | awk '{print $1}'`
    cd $testDir
    rm *.txt
    rm *.inv.gz
    rm *.dtrace
    fixModDir client
    fixModDir server
}

function rmCreated {
 cd $testDir/$1
 rm *.txt
}

function runLogMerger {
 cd $testDir
 mv $1/*.txt ./
 mv $2/*.txt ./
 dinv -v -logmerger *Encoded.txt *Log.txt
}

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
installDinv
instrument client
instrument server
runTestProgram server
runTestProgram client
runLogMerger client server
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanup
