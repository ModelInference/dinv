#!/bin/bash

#this script executes a javalance script on the current directory


DINV=$GOPATH/src/bitbucket.org/bestchai/dinv
tornago=$DINV/examples/ricartagrawala


function genMutants {
    cd /tmp
    rm -r go-mutesting*

    go-mutesting --exec "$GOPATH/src/github.com/zimmski/go-mutesting/scripts/simple.sh" --verbose --debug --do-not-remove-tmp-folder --exec-timeout 1 bitbucket.org/bestchai/dinv/examples/ricartagrawala/


    cd go-mutesting*
    cd home/stewartgrant/go/src/bitbucket.org/bestchai/dinv/examples/ricartagrawala/

    mkdir $tornago/tmp
    mv ./* $tornago/tmp
}

function backupOriginal {
    cd $tornago
    
    #backup the test directory, and the orginal source code
    mkdir backup
    cp ricartagrawala.go backup
    cp -r "test" backup
}

function runOriginal {
    cd $tornago
    #run the orignal code
    mv tmp/*original ricartagrawala.go
    ./test.sh
    #move the populated orginal test to it's orignial dir
    mv "test" original-test
}
    


function runMutants {
    cd $tornago/tmp
    rm -r ../test
    for file in ./*; do
        cp -r ../backup/test ../test
        mv $file ../ricartagrawala.go
        
        echo running tests on $file
        cd ..
        ./test.sh
        cd tmp

        cleanName=`echo $file | sed 's/[:\/]//g'`
        mv ../test ../dir-$cleanName-test
        mv ../ricartagrawala.go ../dir-$cleanName-test
    done
}

function checkInvariants {
    cd $tornago
    for mutant in dir*;do 
        #dinv
       # echo $mutant
        cd original-test
        for traceDir in dinv*; do
        #    echo $traceDir
            if [ -d ../$mutant/$traceDir ]; then
                echo match ../$mutant/$traceDir
                java daikon.tools.InvariantChecker  $traceDir/*.inv ../$mutant/$traceDir/*.dtrace >> ../$mutant/dinv.txt
            else
                :
            fi
        done
        for traceDir in daikon*; do
        #    echo $traceDir
            if [ -d ../$mutant/$traceDir ]; then
                echo match ../$mutant/$traceDir
                java daikon.tools.InvariantChecker  $traceDir/*.inv ../$mutant/$traceDir/*.dtrace  >> ../$mutant/daikon.txt
            else
                :
            fi
        done
        break
    done

}

function cleanInvariants {
    cd $tornago
    for mutant in dir*;do 
        rm $mutant/passfail.stext
        rm $mutant/dinv.txt
        rm $mutant/daikon.txt
    done
}



function tornago {
    genMutants
    backupOriginal
    runOriginal
    runMutants
    checkInvariants
    passfail
    summarizeOutput "daikon"
    summarizeOutput "dinv"

}

function summarizeOutput {
    cd $tornago
    errorrx=" ([0-9])+ errors found in ([0-9,]+)"
    falserx=" ([0-9])+ false positives, out of ([0-9,]+), "
    for mutant in dir*;do 
        let "dinvTFP = 0"
        let "dinvFP = 0"
        let "dinvTE = 0"
        let "dinvE = 0"
        while IFS='' read -r line || [[ -n "$line" ]]; do
            #echo $line
            if [[ $line =~ $errorrx ]]
            then
                #echo errors: "${BASH_REMATCH[1]}" total: "${BASH_REMATCH[2]}"
                errors=`echo "${BASH_REMATCH[1]}" | sed 's/,//g'`
                total=`echo "${BASH_REMATCH[2]}" | sed 's/,//g'`
                dinvTE=$(($dinvTE + $total))
                dinvE=$(($dinvE + $errors))
            elif [[ $line =~ $falserx ]]
            then
                #echo false positive: "${BASH_REMATCH[1]}" total false: "${BASH_REMATCH[2]}"
                fpositive=`echo "${BASH_REMATCH[1]}" | sed 's/,//g'`
                total=`echo "${BASH_REMATCH[2]}" | sed 's/,//g'`
                dinvTFP=$(($dinvTFP + $total))
                dinvFP=$(($dinvFP + $errors))

            fi 
        done < $mutant/$1.txt
        echo $mutant -- $1    - Errors: $dinvE  total statements $dinvTE FalseP: $dinvFP total statements $dinvTFP >> $tornago/output.txt
    done
}

function passfail {
    cd $tornago
    for mutant in dir*;do
        cd $tornago/$mutant 
        if [[ `grep FAIL passfail.stext` == "" ]]
        then
            echo $mutant 0-- PASSED -- >>$tornago/output.txt
        else
            echo $mutant 0-- FAILED -- >>$tornago/output.txt
        fi
    done
~                
}

function cleanup {

    cd $tornago
    rm -r test
    rm -r original-test
    mv backup/ricartagrawala.go ./ricartagrawala.go
    mv backup/test ./test
    rmdir backup
    rm -r tmp
    #    cleanInvariants

}

if [ "$1" == "-c" ];
then
    cleanup
    exit
fi
tornago
if [ "$1" == "-d" ];
then
    exit
fi
#cleanup

