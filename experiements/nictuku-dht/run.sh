#!/bin/bash

nodes=5
startingport=6000
trackerport=6011
nodesWithFiles=2
nodePID=()

#testDir=~/go/src/github.com/nictuku/dht/examples/find_infohash_and_wait
testDir=~/go/src/bitbucket.org/bestchai/dinv/experiements/nictuku-dht
dinvDir=~/go/src/bitbucket.org/bestchai/dinv


torrentFolder=torrents

#torrentFile=12am.torrent
#realfile=12am.mp3
torrentFile=speech.torrent
realfile=speech.mp3

function install {
    echo "installing dinv"
    cd ~/go/src/bitbucket.org/bestchai/dinv
    sudo -E go install 
    echo "installing DHT"
    cd ~/go/src/github.com/nictuku/dht
    sudo -E go install 
}

function createDirs {
    cd $testDir
    for (( i=0; i<$nodes; i++ ))
    do
        mkdir c$i
        #give the torrent file to a subset of clients
        if [ $i -lt $nodesWithFiles ]
        then
            cp $torrentFolder/$torrentFile c$i
            cp $torrentFolder/$realfile c$i
        else
            cp $torrentFolder/$torrentFile c$i
        fi
    done
}

function run {
    cd $testDir
    sleep 1
    #start the clients
    for (( i=0; i<$nodes; i++))
    do
        sleep 1
        #go run main.go --port=600$i -logtostderr -v 5
        #go run main.go --port=600$i -logtostderr -v 5 &
        gnome-terminal -x go run main.go --port=600$i -logtostderr -v 5
        #go run main.go --port=600$i &
    done

    sleep 10
    killem

}

function moveResults {
    for (( i=0; i<$nodes; i++))
    do
        cd $testDir/c$i
        mv *[dg].txt ..
    done
    cd $testDir
    wc *.txt
}

function merge {
    cd $testDir
    dinv -l -plan=SCM -shiviz *g.txt *d.txt
}

function daikon {
    cd $testDir
    for file in ./*.dtrace; do
        java daikon.Daikon $file
    done
}
    


#cleanup functions
function removeDirs {
    cd $testDir
    rm -r c*
}


function cleanup {
    removeDirs
    killem
    $dinvDir/examples/lib.sh clean
}

function killem {
    killall go
    killall main
}
    
    

if [ "$1" == "-c" ];
then
    cleanup
    exit
fi

if [ "$1" == "-k" ];
then
    killem
    exit
fi

if [ "$1" == "-d" ];
then
    install
    run
    #moveResults
    merge
    daikon
fi

if [ "$1" == "-m" ];
then
    merge
    daikon
fi

exit
