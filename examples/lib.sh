#/bin/bash

set -e

function clean {
    baseDir="log_archive"
    if [ "$1" != "" ]; then
        baseDir=$1
    fi
    dir="./$baseDir/$(date +%m-%d_%H-%M-%S)/"

    echo "moving left-overs to $dir"

    mkdir -p "$dir"
    mv ./*{Log,Encoded}.txt Shiviz.log ./*.dtrace ./*.gz ./*.output output.txt $2 \
       -t "$dir" 2>/dev/null || true

    if [ $(find "$dir" -mindepth 1 -maxdepth 1 | wc -l) -eq 0 ]; then
        echo "nothing to clean up"
        rm -r $dir
    else
        ln -s -f "$dir" last_run
        echo "moved left-overs to $baseDir/last_run"
    fi
}

# TODO instrument all function that applies dinv and govec to all arguments

function installDinv {
    echo "compile dinv"
    go install bitbucket.org/bestchai/dinv
}

# first argument to runLogMerger ($1) is passed to dinv
# ex: use to specify merging strategy: runLogMerger '-plan SCM -shiviz'
function runLogMerger {
    t1=$(date +'%s')
    if [ "$1" != "" ]; then
        echo "merge logs with extra args '$1'"
    fi

    dinv -v -l $1 ./*Encoded.txt ./*Log.txt
    # dinv -v -l $1 -name="fruits" -shiviz ./*Encoded.txt ./*Log.txt
    echo "logmerger took $(($(date +'%s') - $t1))s to run"
}

function runDaikon  {
    t1=$(date +'%s')
    echo "run daikon"
    # redirect output both to output.txt and stdout
    java daikon.Daikon ./*.dtrace | tee output.txt
    echo "daikon took $(($(date +'%s') - $t1))s to run"
}

case $1 in
    "clean" )
        clean "$2" "$3"
        ;;
    "installDinv" )
        installDinv
        ;;
    "runLogMerger")
        runLogMerger "$2"
        ;;
    "runDaikon" )
        runDaikon
        ;;
    "" )
        echo "available commands"
        echo "clean [directory to move left-overs to] [list of extra files to include]"
        echo "installDinv"
        echo "runLogMerger [extra dinv arguments, i.e. '-plan SCM -shiviz'"
        echo "runDaikon"
        exit 1
        ;;
esac
