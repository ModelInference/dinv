# linear
 The linear server has three hosts, client, coeff, and linn. Who
 communicate  only with their neighbour
 client <--> coeff <--> linn
 the client sends two random integers (term 1, term2) to coeff. Coeff generates a
 random coefficient and sends all three variables to linn (term1,
 term2, coeff). linn computes (linn = coeff * term1 + term2) and
 sends the result back down the line.

#Files
    * client
        * client.go > generates terms to be sent to the coeff server,
          send them and waits for a response
    * coeff
        * coeff.go > listens for a request from the server, upon
          receiving two terms, it generates a coefficient and sends all
three integers to the server. It then waits for a response from the
server and upon receiving it, propagates the response to the client
    * comm
        * comm.go > communication package that is shared by the three
          hosts
    * linn > calculates the linear equation ` sum = coeff * term1 +
      term 2` based values sent to it from the coeff server. Upon
calculating the sum, it is sent back to coeff.

#Invariants
The variables of interest are `term1`, `term2`, `coeff` and `sum`. The
invariants on these variables correspond to qualities of linear
equations. The following should be detected

`term1 <= sum`
`term2 <= sum`
`coeff <= sum`

This script `run.sh` manages the instrumentation, and execution of the host programs, as well as the merging of the generated logs, and the execution of daikon. 

Default behaviour : Execute, and cleanup

#Options 
   -d : dirty run, all generated files are left after execution for
   inspection
   -c : cleanup, removes generated files created during the run
