#!/bin/bash

dinv -instrumenter $DINV/TestPrograms/t2/client/breakup/client.go
mv $DINV/TestPrograms/t2/client/breakup/client.go $DINV
mv $DINV/TestPrograms/t2/client/breakup/marshall.go $DINV
cd ..
go run clientEntry.go

sleep 1
cd breakup
rm mod*
rm inject.go
cd ../..
mv client.go client/breakup
mv marshall.go client/breakup



