#!/bin/bash

for (( i=0; i < $1; i++)) 
do
    DINVID=$i
    export DINVID
    ./env.sh &
done

