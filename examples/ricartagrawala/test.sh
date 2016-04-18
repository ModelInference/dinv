#!/bin/bash

#ricart-agrawala test cases


function runTest {
    for (( i=0; i<$2; i++))
    do
        go test $1 -id=$i -hosts=$2 &
    done

    sleep 1
    rm *.txt

}

runTest "hoststartup_test.go" 3
