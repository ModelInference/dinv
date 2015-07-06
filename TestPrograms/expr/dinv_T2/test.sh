#!/bin/bash
#this test creates the instrurmented version of the test programs
#and runs them through the entire process from start to finnish
#clean up directory

P1="server.go"
P2="client.go"
TEST="t2"

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function runInstrumenter {
    cd $DINV
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P2 > $DINV/TestPrograms/$TEST/mod_$P2
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P1 > $DINV/TestPrograms/$TEST/mod_$P1
}

function runTestPrograms {
    cd $DINV/TestPrograms/$TEST
    go run mod_$P1 &
    SERVER_PID=$!
    echo $SERVER_PID
    sleep 1
    go run mod_$P2 &
    CLIENT_PID=$!
    echo $CLIENT_PID
    sleep 1
    kill $SERVER_PID
    kill $CLIENT_PID
    kill `ps | pgrep mod_server | awk '{print $1}'`
    kill `ps | pgrep mod_client | awk '{print $1}'`
}

function runLogMerger {
    cd $DINV
    dinv -logmerger $DINV/TestPrograms/$TEST/$P2.txt $DINV/TestPrograms/$TEST/$P1.txt
    mv ./*.dtrace $DINV/TestPrograms/expr/dinv_T3/
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
    rm $DINV/TestPrograms/$TEST/$P1.txt
    rm $DINV/TestPrograms/$TEST/$P2.txt
    rm $DINV/TestPrograms/$TEST/mod_$P1
    rm $DINV/TestPrograms/$TEST/mod_$P2
    rm $DINV/TestPrograms/$TEST/testclient.log-Log.txt
    rm $DINV/TestPrograms/$TEST/slog.log-Log.txt
    rm $DINV/TestPrograms/expr/dinv_T2/*.dtrace
    rm $DINV/TestPrograms/expr/dinv_T2/*.gz
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
