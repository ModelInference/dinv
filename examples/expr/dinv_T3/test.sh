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
    cd $DINV/TestPrograms/$TEST/$P1
    dinv -instrumenter -v $DINV/TestPrograms/$TEST/$P1
    swapMods $P1
    
    cd $DINV/TestPrograms/$TEST/$P2
    dinv -instrumenter -v $DINV/TestPrograms/$TEST/$P2
    swapMods $P2
    
    cd $DINV/TestPrograms/$TEST/$P3
    dinv -instrumenter -v $DINV/TestPrograms/$TEST/$P3
    swapMods $P3
}

function swapMods {
    mkdir ../temp$1
    mv mod* ../temp$1
    mkdir ../temp2$1
    mv *.go ../temp2$1
    mv ../temp$1/*.go ./
    rmdir ../temp$1
}

function fixModDir {
    cd $DINV/TestPrograms/$TEST/temp2$1
    mv *.go ../$1
    cd ..
    rmdir temp2$1

}

function runTestPrograms {
    cd $DINV/TestPrograms/$TEST/$P3
    go run mod_$P3.go &
    sleep 1
    cd $DINV/TestPrograms/$TEST/$P2
    go run mod_$P2.go &
    sleep 1
    cd $DINV/TestPrograms/$TEST/$P1
    go run mod_$P1.go &
    wait $!
    kill `ps | pgrep mod_ | awk '{print $1}'`

    fixModDir $P1
    fixModDir $P2
    fixModDir $P3
    mv $DINV/TestPrograms/$TEST/$P1/*.txt $DINV/
    mv $DINV/TestPrograms/$TEST/$P2/*.txt $DINV/
    mv $DINV/TestPrograms/$TEST/$P3/*.txt $DINV/

}

function runLogMerger {
    cd $DINV
    dinv -v -logmerger *Encoded.txt *Log.txt
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
    rm $DINV/TestPrograms/$TEST/$P1/mod_*.go
    rm $DINV/TestPrograms/$TEST/$P2/mod_*.go
    rm $DINV/TestPrograms/$TEST/$P3/mod_*.go
    cd $DINV
    rm *.txt
    cd $DINV/TestPrograms/expr/dinv_T3
        rm ./*.dtrace
        rm ./*.gz
}



if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi
installDinv
runInstrumenter
runTestPrograms
runLogMerger
shivizMerge
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanUp
