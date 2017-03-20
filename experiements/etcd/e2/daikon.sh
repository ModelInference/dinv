#!/bin/bash
for file in ./*.dtrace; do
    java daikon.Daikon $file
done
