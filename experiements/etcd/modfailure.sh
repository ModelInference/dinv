#!/bin/bash

#Simulated Failure Script

PEERS=$1
LOOPS=1
MENU="k:kill\nw:wake\ns:status\nm:murder\nq:quit\nh:help"


for i in $(seq 1 $PEERS)
do
    DEAD[$i]="ALIVE"
done

echo -e $MENU
while [ 1 ]
do
    read -p "command:" com
    case $com in
        "k" )
            read -p "kill who [1-$PEERS]:" PEER
            if [ ${DEAD[$PEER]} == "ALIVE" ]; then
                echo killing peer 127.0.0.$PEER
                sudo iptables -A INPUT -p tcp -s 127.0.0.$PEER --destination-port 2380 -j DROP
                DEAD[$PEER]="DEAD"
            else
                echo peer 127.0.0.$PEER is dead
            fi
            ;;
        "w" )
            read -p "wake who [1-$PEERS]:" PEER
            if [ ${DEAD[$PEER]} == "DEAD" ] ; then
                FIX=`sudo iptables -L INPUT -n --line-numbers | grep 127.0.0.$PEER | cut -d " " -f 1`
                sudo iptables -D INPUT $FIX
                echo waking up peer 127.0.0.$PEER 
            else
                echo peer 127.0.0.$PEER is allready awake
            fi
                ;;
        "s" )
            for i in $(seq 1 $PEERS)
            do
                echo 127.0.0.$i : ${DEAD[$i]}
            done
            ;;
        "m" )
            killall etcd ;;
        "q" )
            echo exiting
            killall etcd
            exit ;;
        "h" )
            echo -e $MENU ;;
    esac
done


