#!/bin/bash
#doc.sh starts the godoc webserver and opens a web browser to view the DInv documentation.

address=localhost:6060/pkg/bitbucket.org/bestchai/dinv

if hash godoc 2>/dev/null; then
    godoc -http=:6060 &
    server=$!
else
    echo "unable to run godoc, please install and run again"
    exit
fi

sleep 1 #let the server wake up

#check if firefox is available otherwise quit
#TODO make this work for other browsers
browser=""
if hash firefox 2>/dev/null; then
    firefox $address &
    browser=$!
else
    echo "unable to open in browser, download firefox please"
fi

wait $browser
kill $server

