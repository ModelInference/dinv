#G!/bin/bash
#this test creates the instrurmented version of the test programs
#and runs them through the entire process from start to finnish
#clean up directory

P1="server"
P2="client"
TEST=""
testDir=$DINV/examples/sum


function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function instrument {
    dinv -i $testDir/$1/lib
}


function fixModDir {
    rm -r $testDir/$1
    mv $testDir/$1_orig $testDir/$1
}

function runTestProgram {
    cd $testDir/$1
    go run $1Entry.go &
    sleep 1
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
runTestProgram client
runTestProgram server
runLogMerger client server
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanUp
