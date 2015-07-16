#!/bin/bash
#insturments and executes raft

RAFT="$GOPATH/src/github.com/wantonsolutions/raft"

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
    cd $DINV/instrumenter
    go install
}

function runInstrumenter {
    cd $RAFT
    echo "Insturmenting"
    dinv -instrumenter $RAFT/raft.go
}

function runTestPrograms {
    cd $RAFT
    mv raft.go ..
    go test
    mv ../raft.go ./
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
    rm $RAFT/mod_raft.go
}



installDinv
runInstrumenter
runTestPrograms
#runLogMerger
#shivizMerge
#runDaikon
#cleanUp
