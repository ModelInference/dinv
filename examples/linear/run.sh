#!/bin/bash
# linear/run.sh controls the exection of the linear fit example
# server.
# The linear server has three hosts, client, coeff, and linn. Who
# communicate  only with their neighbour
# client <--> coeff <--> linn
# the client sends two random integers (term 1, term2) to coeff. Coeff generates a
# random coefficient and sends all three variables to linn (term1,
# term2, coeff). linn computes (linn = coeff * term1 + term2) and
# sends the result back down the line.

#This script mannages the instrumentation, and execution of the host programs, as well as the merging of the generated logs, and the execution of daikon. 
# The detected invarients should include equality among variables
# (term1, term 2, coeff, linn) across each host, and inequalities such
# as term1 < linn, term2 < linn.

#Default behaviour : Execute, and cleanup

#Options 
#   -d : dirty run, all generated files are left after execution for
#   inspection
#   -c : cleanup, removes generated files created during the run

P1="client"
P2="coeff"
P3="linn"
DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
testDir=$DINV/examples/linear

function installDinv {
    echo "Install dinv"
    cd $DINV
    go install
}

function instrument {
    dinv -i -v -dir=$testDir/$1
    GoVector -v -dir=$testDir/$1
}


function fixModDir {
    if [ -d $testDir/$1 ]; then
        if [ -d $testDir/$1_orig ]; then
            rm -r $testDir/$1
            mv $testDir/$1_orig $testDir/$1
        fi
    fi
}


function runTestPrograms {
    cd $testDir/$P3
    go run $P3.go &
    sleep 1
    cd $testDir/$P2
    go run $P2.go &
    sleep 1
    cd $testDir/$P1
    go run $P1.go &
    sleep 1
    wait $!
    
    mv $testDir/$P1/*.txt $testDir
    mv $testDir/$P2/*.txt $testDir
    mv $testDir/$P3/*.txt $testDir
    
}

function runLogMerger {
    cd $testDir
    dinv -v -l -name="fruits" -plan="SCM" -shiviz *Encoded.txt *Log.txt
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

function shivizMerge {
    cat $rm $DINV/TestPrograms/$TEST/$P1.log-Log.txt $DINV/TestPrograms/$TEST/$P2.log-Log.txt $DINV/TestPrograms/$TEST/$P3.log-Log.txt > $DINV/TestPrograms/expr/dinv_T3/shiviz.txt
    
}


function cleanUp {
    kill `ps | pgrep $P3 | awk '{print $1}'`
    kill `ps | pgrep $P2 | awk '{print $1}'`
    rm ./*.dtrace
    rm ./*.gz
    rm ./*.txt
    fixModDir $P1
    fixModDir $P2
    fixModDir $P3
    
}



if [ "$1" == "-c" ];
then
    cleanUp
    exit
fi
installDinv
instrument $P1
instrument $P2
instrument $P3
runTestPrograms
runLogMerger
shivizMerge
runDaikon
if [ "$1" == "-d" ];
then
    exit
fi
cleanUp
