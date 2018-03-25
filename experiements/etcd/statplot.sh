#!/bin/bash
#collect the number of requests serviced over the course of the execution

#collect bandwidth measure now that the script is done
last=`tail -1 bandwidth.txt`
echo "" > bandwidth.dat
echo $last
for(( i=1 ; 1 < last ; i++)); do
    c=`grep -c "^$i$" bandwidth.txt`
    if [ "$i" == "$last" ]; then
        break
    fi
    echo "$i, $c" >> bandwidth.dat
done

cp bandwidth.dat govectorEval/instrumentedBandwidth.dat
#cp bandwidth.dat govectorEval/unmodifiedBandwidth.dat

cp latency.dat govectorEval/instrumentedLatency.dat
#cp latency.dat govectorEval/unmodifiedLatency.dat

R CMD BATCH summary.R
cat summary.Rout

gnuplot bandwidth.plot
evince bandwidth.pdf


gnuplot latency.plot
evince latency.pdf
