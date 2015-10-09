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

DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/serialize

#Install Dinv to ensure any devloper modifications are up to date
function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

# instrument files based on directory
function instrument {
    dinv -i -v -dir=$testDir/$1
}

# after the instrumenter runs, the original contents of the directory
# will be moved to a folder *originalDirectory*_org. This function
# resets the 
function fixModDir {
    rm -r $testDir/$1
    mv $testDir/$1_orig $testDir/$1
}

# used to run the instrumented test files
function runTestProgram {
    cd $testDir/server
    go run server.go &
    sleep 1
    clients=3
    cd $testDir/client
    for (( i = 0 ; i <clients ;i++))
    do
        go run client.go $i &
    done
}

#cleanup removes the generated log files, and tracke files. Furthermore, it kills the server process to free the port for future executions.
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

#rm Created removes the generated text files in a specified directory. This is used to clean up the logging files
function rmCreated {
 cd $testDir/$1
 rm *Encoded.txt
 rm *Readable.txt
 rm *Log.txt
}

#the log merger is run by passing it the encoded point files and the govector log files generated during the run.
function runLogMerger {
 cd $testDir
 mv $1/*.txt ./
 mv $2/*.txt ./
 dinv -v -l -shiviz *Encoded.txt *Log.txt
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
installDinv
instrument client
instrument server
runTestProgram $1
runLogMerger client server
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanup
