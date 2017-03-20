#!/bin/bash

#cluster args
#arguments
#1 function
#2 clients
#3 name
#4 assertType
#5 leader
#6 sample

TESTS=3
NAME="SL-Sample-100"
#PULL
#./cluster.sh -p
assertOP[0]="NONE"
assertOP[1]="STRONGLEADER"
assertOP[2]="LOGMATCHING"
assertOP[3]="LEADERAGREEMENT"

leaderOP[0]="true"
leaderOP[1]="false"

bugOP[0]="true"
bugOP[1]="false"

sampleOP[0]="1"
sampleOP[1]="10"
sampleOP[2]="100"

clientOP[0]="4"
clientOP[1]="8"
clientOP[2]="12"

for assert in ${assertOP[@]}
do
    for bug in ${bugOP[@]}
    do
        for leader in ${leaderOP[@]}
        do
            for sample in ${sampleOP[@]}
            do
                for client in ${clientOP[@]}
                do
                    for ((i=0; i < TESTS;i++))
                    do
                        ./cluster.sh -r $client $assert-leader-$leader-sample-$sample-client-$client-bug-$bug $assert $leader $sample $bug
                    done
                    #example
                    #./cluster.sh -r 4 STRONGLEADER-leader-true-sample-10-client-4-bug-true STRONGLEADER true 10 true

                done
            done
        done
    done
done


