#!/bin/bash
#this test creates the instrurmented version of the test programs
#and runs them through the entire process from start to finnish
#clean up directory

P1="client"
P2="coeff"
P3="linn"
TEST="t3"

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function runInstrumenter {
    cd $DINV
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P1/$P1.go > $DINV/TestPrograms/$TEST/mod_$P1.go
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P2/$P2.go > $DINV/TestPrograms/$TEST/mod_$P2.go
    dinv -instrumenter $DINV/TestPrograms/$TEST/$P3/$P3.go > $DINV/TestPrograms/$TEST/mod_$P3.go
}

function runTestPrograms {
    cd $DINV/TestPrograms/$TEST
    go run mod_$P3.go &
    sleep 1
    go run mod_$P2.go &
    sleep 1
    go run mod_$P1.go &
    wait $!
    kill `ps | pgrep mod_ | awk '{print $1}'`
    mv $P1/$P1.go.txt $P1.go.txt
    mv $P2/$P2.go.txt $P2.go.txt
    mv $P3/$P3.go.txt $P3.go.txt

}

function runLogMerger {
    cd $DINV
    dinv -logmerger $DINV/TestPrograms/$TEST/$P1.go.txt $DINV/TestPrograms/$TEST/$P2.go.txt $DINV/TestPrograms/$TEST/$P3.go.txt
    #dinv -logmerger $DINV/TestPrograms/$TEST/$P2.go.txt $DINV/TestPrograms/$TEST/$P3.go.txt

    mv ./*.dtrace $DINV/TestPrograms/expr/dinv_T3/
}

function runDaikon {
    cd $DINV/TestPrograms/expr/dinv_T3/
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
    rm output.txt
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
#    clear
    cat output.txt
}

function shivizMerge {
    cat $rm $DINV/TestPrograms/$TEST/$P1.log-Log.txt $DINV/TestPrograms/$TEST/$P2.log-Log.txt $DINV/TestPrograms/$TEST/$P3.log-Log.txt > $DINV/TestPrograms/expr/dinv_T3/shiviz.txt
    
}

function cleanUp {
    rm $DINV/TestPrograms/$TEST/$P1.go.txt
    rm $DINV/TestPrograms/$TEST/$P2.go.txt
    rm $DINV/TestPrograms/$TEST/$P3.go.txt
    rm $DINV/TestPrograms/$TEST/mod_$P1.go
    rm $DINV/TestPrograms/$TEST/mod_$P2.go
    rm $DINV/TestPrograms/$TEST/mod_$P3.go
    rm $DINV/TestPrograms/$TEST/$P1.log-Log.txt
    rm $DINV/TestPrograms/$TEST/$P2.log-Log.txt
    rm $DINV/TestPrograms/$TEST/$P3.log-Log.txt
    cd $DINV/TestPrograms/expr/dinv_T3
        rm ./*.dtrace
        rm ./*.gz
}



installDinv
runInstrumenter
runTestPrograms
runLogMerger
shivizMerge
runDaikon
#cleanUp
