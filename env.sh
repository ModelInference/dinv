#!/bin/bash

ITTS=3

for ((i=0 ; i<$ITTS ; i++ ))
do
    printenv | grep DINVID
    sleep 1
done
