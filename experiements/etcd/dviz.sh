#!/bin/bash
sudo -E go install ~/go/src/github.com/wantonsolutions/Dviz
for file in ./*.json; do
    Dviz $file
done
