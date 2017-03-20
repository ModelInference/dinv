
INFRA=$1
PUBLIC=$2
PRIVATE=$3
CLUSTER=$4
DINV_ASSERT_PEERS=$5

ASSERTTYPE=$6
LEADER=$7
SAMPLE=$8
DINVBUG=$9

HOME=/home/stewart
DINV=$HOME/go/src/bitbucket.org/bestchai/dinv
ETCD=$HOME/go/src/github.com/coreos/etcd
ETCDCMD=$HOME/go/src/github.com/coreos/etcd/bin/etcd

DINV_HOSTNAME=$PUBLIC
DINV_ASSERT_LISTEN=$PRIVATE:12000
#export each of the names before launching a node
export DINV_HOSTNAME
export DINV_ASSERT_PEERS
export DINV_ASSERT_LISTEN

#export assert macros
export LEADER
export ASSERTTYPE
export SAMPLE
export DINVBUG


echo "cmd $ETCDCMD infra $INFRA public $PUBLIC private $PRIVATE cluster $CLUSTER assert $DINV_ASSERT_PEERS bug $" > lastConfig

$ETCDCMD --name infra$INFRA --initial-advertise-peer-urls http://$PRIVATE:2380 \
  --listen-peer-urls http://$PRIVATE:2380 \
  --listen-client-urls http://$PRIVATE:2379,http://127.0.0.1:2379 \
  --advertise-client-urls http://$PRIVATE:2379 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster $CLUSTER \
  --initial-cluster-state new
