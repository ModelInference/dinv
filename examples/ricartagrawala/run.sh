#!/bin/bash

    for (( i=0; i<$1; i++))
    do
        go test $2 -id=$i -hosts=$1 &
    done

    sleep 1
    rm *.txt
