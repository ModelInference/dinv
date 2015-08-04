#!/bin/bash


function documentDir {
for i in $1/* ; do
    if [ -d "$i" ]; then
        echo $i
        if [ "$i" != $DINV/doc ]; then
            documentDir $i
        fi
    elif [ -f "$i" ]; then
        for gofiles in $i
        echo $i
    fi
done
}


mkdir $DINV/doc
documentDir $DINV
