#!/bin/bash
#./blast.sh words.txt serverIpPort command id
# $1 textfile to put and get from the server
# $2 serverIpPort
# $3 The command to execute (etcd client binary)
# $4 unique Id this client marks files with
echo "starting azure client"
lout=latency$4.txt
echo "" > $lout
i=0
for word in $(<$1)
do
    #measure the latency in miliseconds
    a=$(($(date +%s%N)/1000000))
    ETCDCTL_API=3 $3 --endpoints=$2:2379 put $i "$word"
    b=$(($(date +%s%N)/1000000))
    lat=0
    let lat=b-a
    echo $lat >> $lout

#    ETCDCTL_API=3 ../bin/etcdctl --endpoints=localhost:2379 get $i
    i=$((i+1))
done


#i=0
#for word in $(<$1)
#do
#    ETCDCTL_API=3 $3 --endpoints=$2:2379 get $i
#    i=$((i+1))
#done
