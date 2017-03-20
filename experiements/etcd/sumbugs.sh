#!/bin/bash

output=bs.txt
echo "" > $output
for file in bugstart*; do
   cat $file >>$output
   #rm $file
   echo "" >> $output
done
#rm $output
START=`sort $output | head -2`
#get the earliest bug catching time
output=bc.txt
echo "" > $output
for file in bugcatch*; do
   cat $file >>$output
   #rm $file
   echo "" >> $output
done
CATCH=`sort $output | head -2`
DIFF=`echo $CATCH - $START | bc`
echo $DIFF


