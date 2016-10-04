#!/bin/bash
set -e

# http://unix.stackexchange.com/a/145654
exec &> >(tee -a "run.output")

N=3
M=100

function runSystem {
    echo "run system with $N nodes, and $M counter increments"

    nodes=()
    for n in $(seq 1 $N); do
        # start nodes; wrap around at the end
        DINV_HOSTNAME="N800$n" ./ring -port "800$n" -neighbor "800$((n % N + 1))" &
        nodes[$((n-1))]="$!"
    done

    sleep 1

    for m in $(seq 1 $M); do
        # select one node after the other
        pid="${nodes[$((m % N))]}"
        echo "sending SIGUSR1 to N800$((m % N))"
        kill -SIGUSR1 "${nodes[$((m % N))]}"
        sleep 0.2
    done

    # terminate all nodes
    for job in $(jobs -p); do
        kill $job
    done
}


if [ "$1" != "" ]; then
    N="$1"
fi
if [ "$2" != "" ]; then
    M="$2"
fi

pushd "$(dirname "$0")"

../lib.sh clean "failed"
../lib.sh installDinv
# GoVector -dir "client"
# dinv -i -file "client/client.go"
# GoVector -dir "server"
# dinv -i -file "server/server.go"
go build
runSystem
../lib.sh runLogMerger '-plan SRM'
../lib.sh runDaikon
../lib.sh clean

popd
