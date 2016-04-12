#!/bin/bash

    for (( i=0; i<$1; i++))
    do
        go run ricart-agrawala.go $i $1 &
    done

    sleep 1
    rm *.txt
