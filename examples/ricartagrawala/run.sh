#!/bin/bash

    for (( i=0; i<$1; i++))
    do
        go test -id=$i -hosts=$1 &
    done

    sleep 1
    rm *.txt
