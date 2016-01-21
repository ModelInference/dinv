#!/bin/bash
# diningPhil/run.sh controls the exection of the dining philosophers
# example diningPhilosophers runs on an arbetrary number of hosts, the
# communication pattern follows Host_i-1 <--> Host_i <--> Host_i+1
# That is, every host has a neighbour, and only communicates with that
# neighbour

Hosts=5
BasePort=4000

function runTestPrograms {
    for (( i=0; i<Hosts; i++))
    do
        let "hostPort=i + BasePort"
        let "neighbourPort= (i+1)%Hosts + BasePort"
        go run diningphilosophers.go -mP $hostPort -nP $neighbourPort &
    done
}

runTestPrograms
    kill `ps | pgrep dining | awk '{print $1}'`
