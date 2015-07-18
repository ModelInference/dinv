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
    cd $DINV/TestPrograms/$TEST/$P2/breakup/
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P2/breakup $P2
    mkdir ../temp
    mv mod* ../temp
    mkdir ../temp2
    mv *.go ../temp2
    mv ../temp/*.go ./
    rmdir ../temp
    
    cd $DINV/TestPrograms/$TEST/$P1/breakup/
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P1/breakup $P1
    mkdir ../temp
    mv mod* ../temp
    mkdir ../temp2
    mv *.go ../temp2
    mv ../temp/*.go ./
    rmdir ../temp
}

function runTestPrograms {
    cd $DINV/TestPrograms/$TEST/$P1
    go run serverEntry.go &
    SERVER_PID=$!
    echo $SERVER_PID
    sleep 1
    cd $DINV/TestPrograms/$TEST/$P2
    go run clientEntry.go &
    CLIENT_PID=$!
    echo $CLIENT_PID
    sleep 1
    kill $SERVER_PID
    kill $CLIENT_PID
    kill `ps | pgrep serverEntry | awk '{print $1}'`
    kill `ps | pgrep clientEntry | awk '{print $1}'`
    
    cd $DINV/TestPrograms/$TEST/$P2/temp2
    mv *.go ../breakup
    cd ..
    rmdir temp2
    
    cd $DINV/TestPrograms/$TEST/$P1/temp2
    mv *.go ../breakup
    cd ..
    rmdir temp2
    
    mv $DINV/TestPrograms/$TEST/$P2/*.txt $DINV
    mv $DINV/TestPrograms/$TEST/$P1/*.txt $DINV
}

function runLogMerger {
    cd $DINV
    dinv -logmerger client-*.txt server-*.txt
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
    rm $DINV/testclient.log-Log.txt
    rm $DINV/slog.log-Log.txt
    $DINV/TestPrograms/expr/dinv_T2/*.dtrace
    rm $DINV/TestPrograms/expr/dinv_T2/*.gz
    rm $DINV/TestPrograms/expr/dinv_T2/output.txt
    
    cd $DINV/TestPrograms/$TEST/$P1/breakup/
    rm mod*
    cd .. 
    rm *.txt
    cd $DINV/TestPrograms/$TEST/$P2/breakup/
    rm mod*
    rm *.txt
    cd $DINV 
    rm client-*.txt server-*.txt
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
