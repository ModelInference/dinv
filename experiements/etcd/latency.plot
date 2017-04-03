set term pdf
set output "latency.pdf"

#plot "latency.dat" u 1:2:(10) smooth kdensity t "latency"
#plot "latency.dat" smooth kdensity t "latency"
#plot "latency.dat" u 1:2 smooth bezier
#plot "latency.dat" u 1:2 smooth csplines
#plot "latency.dat" u 1:2 smooth acsplines
#plot "latency.dat" u 1:2 smooth unique
#plot "latency.dat" u 1:2 smooth frequency

#
# This script demonstrates the use of assignment operators and
# sequential expression evaluation to track data points as they
# are read in.
#
# We use the '=' and ',' operators to track the running total
# and previous 5 values of a stream of input data points.
#
# Ethan A Merritt - August 2007
#
# Define a function to calculate average over previous 5 points
#
set title \
    "Go Vector latency\n"
set key invert box center right reverse Left
set xtics nomirror
set xlabel "time (ms)"
set ytics nomirror
set xlabel "latency (ms)"
set border 3

samples(x) = $0 > 4 ? 5 : ($0+1)
avg5(x) = (shift5(x), (back1+back2+back3+back4+back5)/samples($0))
shift5(x) = (back5 = back4, back4 = back3, back3 = back2, back2 = back1, back1 = x)

#
# Initialize a running sum
#
init(x) = (back1 = back2 = back3 = back4 = back5 = sum = 0)

#
# Plot data, running average and cumulative average
#

datafile = 'latency.dat'

instLatency = 'govectorEval/instrumentedLatency.dat'
unmodifiedLatency = 'govectorEval/unmodifiedLatency.dat'
#set xrange [0:57]

set style data linespoints

#plot sum = init(0), \
#     datafile using 0:2 title 'data' lw 2 lc rgb 'forest-green', \
#     '' using 0:(avg5($2)) title "running mean over previous 5 points" pt 7 ps 0.5 lw 1 lc rgb "blue", \
#     '' using 0:(sum = sum + $2, sum/($0+1)) title "cumulative mean" pt 1 lw 1 lc rgb "dark-red"

#plot sum = init(0), \
#     datafile using 0:(sum = sum + $2, sum/($0+1)) title "cumulative mean" pt 1 lw 1 lc rgb "dark-red"

#plot sum = init(0), \
#     unmodifiedLatency using 0:(sum = sum + $2, sum/($0+1)) title "cumulative mean" pt 1 lw 1 lc rgb "dark-red", \
#     instLatency using 0:(sum = sum + $2, sum/($0+1)) title "cumulative mean" pt 1 lw 1 lc rgb "dark-green", \

plot sum = init(0), \
      unmodifiedLatency using 0:2 title 'etcd' lw 2 lc rgb 'blue', \
      instLatency using 0:2 title 'etcd with GoVector' lw 2 lc rgb 'red'
