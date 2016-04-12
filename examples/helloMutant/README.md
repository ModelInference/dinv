# helloDinv

helloDinv is the introductory program for understanding and running
DInv. The example involves two separate go programs `client.go` and
`server.go`. The client sends the message `Hello Dinv!` to the server,
and the server responds with the time.

We suggest using hello Dinv to test your installation by running
`./run.sh`. The script instruments the two files, runs the modified
files, mergers the logs, and feeds the generated trace files into
daikon.

#Files
    * client
        * client.go > sends hello to the server and awaits the time as
          a response. dump statements are placed at both the send and
receiving points
    * server
        * server.go > waits for a message from the client, responds
          with the time. dump statements are present at both the
sending and receiving functions.


#Invariants
The variables of interest in this system are the messages being
passes ed around. The only invariant that should be detected is the
equality of messages on both the client and server.

#Work flow
The work flow of instrumenting -> executing -> log merging -> invariant
detection, is documented within the `run.sh` script.
The detected invariant are the messages sent between the client and server.

Default behaviour : Execute, and cleanup

Options 
   -d : dirty run, all generated files are left after execution for
   inspection
   -c : cleanup, removes generated files created during the run
