#!/bin/bash
#this test creates the instrurmented version of the test programs
#and runs them through the entire process from start to finnish
#clean up directory




cd $DINV/instrumenter
instrumenter $DINV/TestPrograms/t1/assignment1.go > $DINV/TestPrograms/t1/assignment1_mod.go
instrumenter $DINV/TestPrograms/t1/serverUDP.go > $DINV/TestPrograms/t1/serverUDP_mod.go

cd $DINV/TestPrograms/t1
go run serverUDP_mod.go &
SERVER_PID=$!
echo $SERVER_PID
sleep 1
go run assignment1_mod.go &
CLIENT_PID=$!
echo $CLIENT_PID
sleep 1


kill $SERVER_PID
kill $CLIENT_PID

kill `ps | pgrep serverUDP_mod | awk '{print $1}'`

cd $DINV/logmerger
logmerger $DINV/TestPrograms/t1/assignment1.go.txt $DINV/TestPrograms/t1/serverUDP.go.txt
mv daikonLog.dtrace $DINV/TestPrograms/expr/dinv_T1/

cd $DINV/TestPrograms/expr/dinv_T1/
java daikon.Daikon daikonLog.dtrace
java daikon.PrintInvariants daikonLog.inv.gz > output.txt

rm $DINV/TestPrograms/t1/assignment1.go.txt
rm $DINV/TestPrograms/t1/assignment1_mod.go
rm $DINV/TestPrograms/t1/serverUDP.go.txt
rm $DINV/TestPrograms/t1/serverUDP_mod.go
rm $DINV/TestPrograms/t1/testclient.log-Log.txt
rm $DINV/TestPrograms/t1/testlog.log-Log.txt


clear
cat output.txt
