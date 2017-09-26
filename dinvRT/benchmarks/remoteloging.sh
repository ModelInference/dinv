#!/bin/bash 
#This script benchmarks the dinv logging api speed. The
#log server resides in
#github.com/wantonsolutions/dara/servers/logserver. This test crosses
#packages and is therefore a bit problematic but nessisary for regression.

#testing parameters are varried and the output is written to records
#with the commit and date of both repositories.

#get commits
CURRENT=`pwd`
cd $GOPATH/src/github.com/wantonsolutions/dara
DARACOMMIT=`git rev-parse HEAD`
cd $GOPATH/src/bitbucket.org/bestchai/dinv
DINVCOMMIT=`hg log --limit 1 | head -1 | cut -f 3 -d:`
cd $CURRENT
DATE=`date +%Y-%m-%d:%H:%M:%S`

Filename="$DATE-$DARACOMMIT-$DINVCOMMIT"
echo $DARACOMMIT
echo $DINVCOMMIT

function benchmark {
    TIME=`go run clientBenchmark.go $1 $2 $3 $4 $5 $6`
    echo "$1,$2,$3,$4,$5,$6,$TIME" >> $Filename
}



#Independent parameters
#Number of Variable Types (Int, float, bool, byte array)
#Size of Byte array

#Together these variables should indicate serialization speeds to a decent degree

#Kill all other instances of dara running on this machine. This is a bit overkill because the name server will also die
killall servers

##If the service is being run on cerf then the name server will need to be restarted
if [ `hostname` == "cerf" ]; then
    "restarting the name server"
    servers -n &
fi

#Install and start the logserver
echo "installing dara servers"
go install github.com/wantonsolutions/dara/servers

#start the log server
servers -l >> logs/serverRecords.log &

export DINV_HOSTNAME="localhost:6969"
export DINV_LOG_STORE="localhost:17000"
export DINV_PROJECT="benchmark"

#int float bool byte bytesize runs

#Arrays for testing a variety of configurations
INT[0]="1"
INT[1]="10"
INT[2]="100"
FLOAT[0]="1"
FLOAT[1]="10"
FLOAT[2]="100"
BOOL[0]="1"
BOOL[1]="10"
BOOL[2]="100"
BYTE[0]="1"
BYTE[1]="10"
BYTE[2]="100"
BYTESIZE[0]="1"
BYTESIZE[1]="100"
BYTESIZE[2]="10000"
EXECUTIONS[0]="1"
EXECUTIONS[1]="10"
EXECUTIONS[2]="100"

#SCALEUP
function SCALEUP() {
    for i in {0..2};
    do
        for E in ${EXECUTIONS[@]}
        do
            benchmark ${INT[i]} ${FLOAT[i]} ${BOOL[i]} ${BYTE[i]} ${BYTESIZE[i]} $E
        done
    done
}

#ALL
function ALL() {
    for I in ${INT[@]}
    do
        for F in ${FLOAT[@]}
        do
            for B in ${BOOL[@]}
            do
                for BY in ${BYTE[@]}
                do
                    for BS in ${BYTESIZE[@]}
                    do
                        for E in ${EXECUTIONS[@]}
                        do
                            benchmark $I $F $B $BY $BS $E
                        done
                    done
                done
            done
        done
    done
}

#run the test
SCALEUP
