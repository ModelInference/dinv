# sum

./run.sh controls the execution of the sum client server program.
client sends two random integers to server, and server responds with
the sum.

This script manages the instrumentation and execution of these
programs, as well as the merging of their generated logs, and
execution of daikon on their generated trace files.

# Files
    * client
        * clientEntry.go _(main package for running client )
        * lib
            * client.go _(generates terms, listens for response
              **instrumented**)
            * marshall.go _(marshing functions for net traffic )
    * server
        * serverEntry.go _(main package for running server)
        * lib
            * server.go _(adds terms, responds with sum
              **instrumented**)
            * marshall.go
    * run.sh    _(scripted execution of client and server
      **instrumented**)

#Invariant
The detected data invariants should include 
`term1 + term2 = sum`
`term1 <= sum`
`term2 <= sum`

##Options 
    * -d : dirty run, all generated files are left after execution for inspection
    * -c : cleanup, removes generated files created during the run
