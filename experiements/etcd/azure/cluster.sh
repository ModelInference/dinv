#!/bin/bash

#Author Stewart Grant
#Feb 2016
#FSE deadline push

#This script mannages running etcd clusters on azure. The clusters are
#run in vm's which are ssh'd into. The names of all of the VMs are
#abbrivated below #Etcd is launched along with a client that makes puts
#and gets. After a predetermined ammount of time the cluster is killed
#and the logs are retrieved via scp.

#arguments
#1 function
#2 clients
#3 name
#4 assertType
#5 leader
#6 sample
#7 bug




#VM's their public and private IP's
#stewart-test-1
ST1G=52.228.27.112
ST1P=10.0.1.4
#stewart-test-2
ST2G=52.228.32.101
ST2P=10.0.1.5
#stewart
S1G=13.64.239.61
S1P=10.0.0.4
#stewart2
S2G=13.64.247.122
S2P=10.0.0.5
#stewart3
S3G=13.64.242.139
S3P=10.0.0.6
#stewart4
S4G=13.88.179.170
S4P=10.0.0.8
#stewartbig
SBG=13.64.149.118
SBP=10.0.0.7


#map a subset of the VM's to the current cluster
GLOBALS1=$S2G
LOCALS1=$S2P
GLOBALS2=$S3G
LOCALS2=$S3P
GLOBALS3=$S4G
LOCALS3=$S4P

#GLOBAL information true of all VMs
HOMEA=/home/stewart
DINV=$HOMEA/go/src/bitbucket.org/bestchai/dinv
DINV_ETCD_AZURE=$DINV/experiements/etcd/azure
ETCD=$HOMEA/go/src/github.com/coreos/etcd
ETCDCMD=$HOMEA/go/src/github.com/coreos/etcd/bin/etcd
ETCDCTL=$HOMEA/go/src/github.com/coreos/etcd/bin/etcdctl
AZURENODE=/dinv/azure/node.sh
#different clients
#CLIENT=/dinv/azure/blast.sh
CLIENT=/benchmarkClient.sh
#CLIENT=/dinv/azure/client.sh
CLIENTMGR=/measure.sh

BENCHMARK="YCSB-A"
#BENCMARK="YCSB-A"
#BENCMARK="YCSB-B"
#BENCMARK="PUT-THEN-GET"
#BENCMARK="PUT-AND-GET"

#LOCAL
DINVDIR=/home/stewartgrant/go/src/bitbucket.org/bestchai/dinv

#MEASUREMENT INFO
MEASURE=true


#TEXT=$ETCD/dinv/kahn.in
#TEXT=/usr/share/dict/words
TEXT=$ETCD/dinv/in.in


USAGE="USAGE\n-k kill all nodes in the cluster\n-p pull from etcd-dinv repo\n-l logmerger\n-d Daikon\n-c clean"

function onall {
    ssh stewart@$GLOBALS1  -x $1  &
    ssh stewart@$GLOBALS2  -x $1  &
    ssh stewart@$GLOBALS3  -x $1  &
    ssh stewart@$SBG -x $1  &
    #ssh stewart@$ST1G -x $1 &
    #ssh stewart@$ST2G -x $1 &
    #ssh stewart@$SG -x $1 &
    #ssh stewart@$S2G -x $1 &
    #ssh stewart@$S3G -x $1 &
}


#kill all the nodes
if [ "$1" == "-k" ];then
    echo kill
    onall "killall etcd; killall blast.sh"
    exit
fi

#have all the nodes pull new code
if [ "$1" == "-p" ];then
    echo clean
    $DINVDIR/examples/lib.sh clean
    echo push
    cd $ETCD
    git add --all
    git commit -m "updating raft for peers"
    git push
    cd $DINV_ETCD_AZURE
    echo pull
    onall "cd $ETCD && git pull && ./build ; cd $DINV && hg pull && hg update"
    exit
fi

#run logmerger
if [ "$1" == "-l" ];then
    sudo -E go install ../../../../../bitbucket.org/bestchai/dinv
    $DINVDIR/examples/lib.sh runLogMerger "-plan=SCM -shiviz"
    exit
fi

#run daikon
if [ "$1" == "-d" ];then
    $DINVDIR/examples/lib.sh runDaikon
    exit
fi

#clean
if [ "$1" == "-c" ];then
    echo clean
    onall "cd; rm *.txt"
    onall "killall etcd"
    onall "killall blast"
    $DINVDIR/examples/lib.sh clean
    rm *.txt
    exit
fi


if [ "$1" == "-r" ];then
    echo run

    onall "cd; pwd; rm bug*"
    onall "rm agg*"
    onall "rm bandwidth*"
    onall "rm latency*"
    onall "rm request*"

    #Example execute ssh 
    #ssh stewart@13.64.239.61 -x "mkdir test"

    #Example execute scp
    #scp stewart@13.64.239.61:/home/stewart/azureinstall.sh astest

    #LOCAL CLUSTER
    CLUSTER="infra0=http://$LOCALS1:2380,infra1=http://$LOCALS2:2380,infra2=http://$LOCALS3:2380"
    ASSERT="$LOCALS1:12000,$LOCALS2:12000,$LOCALS3:12000"
    echo "ssh stewart@$GLOBALS1 -x $ETCD$AZURENODE 0 $GLOBALS1 $LOCALS1 $CLUSTER $ASSERT $4 $5 $6 $7"
    ssh stewart@$GLOBALS1 -x "$ETCD$AZURENODE 0 $GLOBALS1 $LOCALS1 $CLUSTER $ASSERT $4 $5 $6 $7" &
    ssh stewart@$GLOBALS2 -x "$ETCD$AZURENODE 1 $GLOBALS2 $LOCALS2 $CLUSTER $ASSERT $4 $5 $6 $7" &
    ssh stewart@$GLOBALS3 -x "$ETCD$AZURENODE 2 $GLOBALS3 $LOCALS3 $CLUSTER $ASSERT $4 $5 $6 $7" &

    #GLOBAL CLUSTER3

    #CLUSTER="infra0=http://$GLOBALS1:2380,infra1=http://$GLOBALS2:2380,infra2=http://$GLOBALS3:2380"
    #ASSERT="$GLOBALS1:12000,$GLOBALS2:12000,$GLOBALS3:12000"
    #ssh stewart@$GLOBALS1 -x "$ETCD$AZURENODE 0 $GLOBALS1 $GLOBALS1 $CLUSTER $ASSERT" &
    #ssh stewart@$GLOBALS2 -x "$ETCD$AZURENODE 1 $GLOBALS2 $GLOBALS2 $CLUSTER $ASSERT" &
    #ssh stewart@$GLOBALS3 -x "$ETCD$AZURENODE 2 $GLOBALS3 $GLOBALS3 $CLUSTER $ASSERT" &
    sleep 5

    if [ "$MEASURE" = false ] ; then
        #run the client on on the same node it's sending to
        ssh stewart@$GLOBALS1 -x "echo $ETCD/dinv/azure/client.sh $TEXT $LOCALS1 && $ETCD/dinv/azure/client.sh $TEXT $LOCALS1 $ETCDCTL" &
        #kill allthe hosts
        sleep 10
        echo kill
        onall "killall etcd"
        onall "killall blast"
    fi

    #run the client on a node seperate from the one receving
    #ssh stewart@$GLOBALS1 -x "$ETCD$CLIENT $TEXT $LOCALSS2" &

    if [ "$MEASURE" = true ] ; then

        EXP=$3
        RUNTIME=10
        CLIENTS=$2
        #run the client locally
        #./measure.sh /usr/share/dict/words $GLOBALS1:2379 $RUNTIME
        echo "STARTING CLIENT"
        
        #ssh stewart@$GLOBALS1 -x "echo $ETCD$CLIENT $TEXT $LOCALS1 && $ETCD$CLIENT $TEXT $LOCALS1 $ETCDCTL"
        ##BLOCK HERE
        ssh stewart@$SBG -x "echo $DINV_ETCD_AZURE$CLIENTMGR $TEXT $LOCALS1 && $DINV_ETCD_AZURE$CLIENTMGR $TEXT $LOCALS1 $RUNTIME $CLIENTS $ETCDCTL $DINV_ETCD_AZURE$CLIENT $BENCHMARK"
        #kill allthe hosts
        echo kill
        rm agg*
        rm bandwidth*
        rm request*
    fi

    #wait for the test to run

    scp stewart@$GLOBALS1:/home/stewart/*.txt ./
    scp stewart@$GLOBALS2:/home/stewart/*.txt ./
    scp stewart@$GLOBALS3:/home/stewart/*.txt ./
    scp stewart@$SBG:/home/stewart/*.txt ./
    ssh stewart@$SBG -x "rm lat*"
    echo DONE!

    if [ "$MEASURE" = true ] ; then
        #get the latency from the requests
        cat latency* > agglat.txt
        rm latency*
        R -q -e "x <- read.csv('agglat.txt', header = F); summary(x); sd(x[ , 1])" > latstats.txt

        #calculate the bandwidth
        cat bandwidth* > aggband.txt
        #remove whitespace
        sed -i '/^$/d' aggband.txt
        #get the digit on the last line, so we have a termination limit
        last=`tail -1 aggband.txt`
        echo "" > bandwidth.dat
        echo $last
        for(( i=1 ; i < last ; i++)); do
            echo $i
            #count each number
            c=`grep -c "^$i$" aggband.txt`
            if [ "$i" == "$last" ]; then
                break
            fi
            echo "$c" >> bandwidth.dat
        done

        R -q -e "x <- read.csv('bandwidth.dat', header = F); summary(x); sd(x[ , 1])" > bandstats.txt
        

        #get the bug catching times
        #get the earliest bug starting time
        output=bs.txt
        echo "" > $output
        for file in bugstart*; do
           cat $file >>$output
           rm $file
           echo "" >> $output
        done
        START=`sort $output | head -2`
        rm $output
        #get the earliest bug catching time
        output=bc.txt
        echo "" > $output
        for file in bugcatch*; do
           cat $file >> $output
           rm $file
           echo "" >> $output
        done
        CATCH=`sort $output | head -2`
        rm $output
        echo $CATCH - $START
        BUGTIME=`echo $CATCH - $START | bc`

        LATMEDIAN=`grep Median latstats.txt |cut -d: -f2`
        LATMEAN=`grep Mean latstats.txt |cut -d: -f2`
        LATSD=`grep "\[1\]" latstats.txt |cut -d' ' -f2`

        BANDMEDIAN=`grep Median bandstats.txt |cut -d: -f2`
        BANDMEAN=`grep Mean bandstats.txt |cut -d: -f2`
        BANDSD=`grep "\[1\]" bandstats.txt |cut -d' ' -f2`
        #./client.sh /usr/share/dict/words $GLOBALS1:2379
        TP=`grep -E '[0-9]' agg.txt | wc -l | cut -f1`
        let "RPS=$TP/$RUNTIME"


        echo "$EXP,$CLIENTS,$RPS,$LATMEDIAN,$LATMEAN,$LATSD,$BANDMEDIAN,$BANDMEAN,$BANDSD,$BUGTIME" >> measurements.dat
    fi
    exit
fi

echo -e $USAGE
