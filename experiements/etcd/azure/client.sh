#client makes requets to a raft webserver and keeps track of the latency of each request

#client.sh intput.txt [server ip:port]
LatOut=latency.txt
echo "" > $LatOut
for word in $(<$1)
do
    echo $word
    echo $1 $2 $3 $4
    a=$(($(date +%s%N)/1000000))
    ETCDCTL_API=3 $3 --endpoints=$2:2379 put $i "$word"
    b=$(($(date +%s%N)/1000000))
    latency=0
    let latatency=b-a
    echo $latency
    #echo "$latency" >> $LatOut
    #echo "making request"
    #ETCDCTL_API=3 $3 --endpoints=$2 put $i "$word"

done
