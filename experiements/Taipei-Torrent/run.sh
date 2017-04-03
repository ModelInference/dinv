#!/bin/bash

nodes=5
startingport=6000
trackerport=6011
nodesWithFiles=2
nodePID=()

#testDir=~/go/src/github.com/jackpal/Taipei-Torrent/dinv
testDir=~/go/src/bitbucket.org/bestchai/dinv/experiements/Taipei-Torrent
dinvDir=~/go/src/bitbucket.org/bestchai/dinv
torrentFolder=torrents

#torrentFile=12am.torrent
#realfile=12am.mp3
torrentFile=speech.torrent
realfile=speech.mp3

function install {
    $dinvDir/examples/lib.sh installDinv
    echo "installing Taipei-Torrent"
    sudo -E go install github.com/jackpal/Taipei-Torrent
    echo "installing DHT"
    sudo -E go install github.com/nictuku/dht
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
    #start the tracker
    #gnome-terminal -x  Taipei-Torrent --useDHT --createTracker=:$trackerport $torrentFolder/$torrentFile
    sleep 1

    #start the clients
    for (( i=0; i<$nodes; i++))
    do
        cd $testDir/c$i
        Taipei-Torrent --port=600$i -v=3 --useUPnP=false -useLPD=true --useDHT --trackerlessMode=true $torrentFile &
        #Taipei-Torrent --port=600$i --useDHT $torrentFile &
    done

}

function moveResults {
    for (( i=0; i<$nodes; i++))
    do
        mv $testDir/c$i/*[dg].txt $testDir
    done
    wc *.txt
}


#cleanup functions
function removeDirs {
    cd $testDir
    rm -r c*
}


function killem {
    killall gocode
    killall Taipei-Torrent
}
    
    

if [ "$1" == "-c" ];
then
    removeDirs
    killem
    $dinvDir/examples/lib.sh clean
    exit
fi

if [ "$1" == "-k" ];
then
    killem
    exit
fi

if [ "$1" == "-r" ];
then
install
createDirs
run
fi

if [ "$1" == "-d" ];
then
install
moveResults
    $dinvDir/examples/lib.sh runLogMerger
    $dinvDir/examples/lib.sh runDaikon
fi
exit
