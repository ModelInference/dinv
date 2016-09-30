#!/bin/bash
# sum/run.sh controls the execution of the sum client server program. client
# sends two random integers to server, and server responds with the sum.

# This script mannages the instrumentation and execution of these programs, as
# well as the merging of their generated logs, and execution of daikon on their
# generated trace files.

# The detected data invarients should include term1 + term2 = sum

set -e

# http://unix.stackexchange.com/a/145654
exec &> >(tee -a "run.output")

function fixModDir {
    if [ -d "$testDir/$1/"lib_orig ]; then
        rm -r $testDir/$1/lib
        mv $testDir/$1/lib_orig $testDir/$1/lib
    fi
}

function runTestPrograms {
    go run server/server.go &
    sleep 1
    go run client/client.go &
    wait $!
    killall server
}

pushd "$(dirname "$0")"

../lib.sh clean "failed"
../lib.sh installDinv
# GoVector -dir "client"
# dinv -i -file "client/client.go"
# GoVector -dir "server"
# dinv -i -file "server/server.go"
runTestPrograms
../lib.sh runLogMerger
../lib.sh runDaikon
../lib.sh clean

popd
