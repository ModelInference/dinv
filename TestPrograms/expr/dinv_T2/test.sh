#!/bin/bash
#this test creates the instrurmented version of the test programs
#and runs them through the entire process from start to finnish
#clean up directory

P1="server"
P2="client"
TEST="t2"

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function runInstrumenter {
    cd $DINV
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P2/$P2.go
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P1/$P1.go
}

function runTestPrograms {
    cd $DINV
    go run mod_$P1.go &
    SERVER_PID=$!
    echo $SERVER_PID
    sleep 1
    go run mod_$P2.go &
    CLIENT_PID=$!
    echo $CLIENT_PID
    sleep 1
    kill $SERVER_PID
    kill $CLIENT_PID
    kill `ps | pgrep mod_server | awk '{print $1}'`
    kill `ps | pgrep mod_client | awk '{print $1}'`
    mv $DINV/TestPrograms/$TEST/$P2/$P2.go.txt $DINV
    mv $DINV/TestPrograms/$TEST/$P1/$P1.go.txt $DINV
}

function runLogMerger {
    cd $DINV
    dinv -logmerger $P2.go.txt $P1.go.txt
    mv ./*.dtrace $DINV/TestPrograms/expr/dinv_T2/
}

function runDaikon {
    cd $DINV/TestPrograms/expr/dinv_T2/
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
    clear
    cat output.txt
}

function cleanUp {
    rm $DINV/$P1.go.txt
    rm $DINV/$P2.go.txt
    rm $DINV/mod_$P1.go
    rm $DINV/mod_$P2.go
    rm $DINV/testclient.log-Log.txt
    rm $DINV/slog.log-Log.txt
    rm $DINV/TestPrograms/expr/dinv_T2/*.dtrace
    rm $DINV/TestPrograms/expr/dinv_T2/*.gz
    rm $DINV/TestPrograms/expr/dinv_T2/output.txt
}

function shivizMerge {
    cat $DINV/TestPrograms/$TEST/slog.log-Log.txt $DINV/TestPrograms/$TEST/testclient.log-Log.txt > ~/Research/expr/dinv_T2/shiviz.txt
    
}


installDinv
runInstrumenter
runTestPrograms
runLogMerger
shivizMerge
runDaikon
cleanUp
