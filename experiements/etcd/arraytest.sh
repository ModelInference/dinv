#!/bin/bash
assertOP[0]="NONE"
assertOP[1]="STRONGLEADER"
assertOP[2]="LOGMATCHING"
assertOP[3]="LEADERAGREEMENT"

leaderOP[0]="true"
leaderOP[1]="false"

sampleOP[0]="1"
sampleOP[1]="10"
sampleOP[2]="100"

clientOP[0]="4"
clientOP[1]="40"
clientOP[2]="400"

for assert in ${assertOP[@]}
do
    for leader in ${leaderOP[@]}
    do
        for sample in ${sampleOP[@]}
        do
            for client in ${clientOP[@]}
            do
                echo "$assert $leader $sample $client"

            done
        done
    done
done


