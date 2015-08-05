# DInv: distributed invariant detector

State in a distributed system is not easily accessible and must be pieced together from the state at the individual nodes. Developers have few tools to help them extract, check, and reason about distributed state.

DInv is a suite of tools to (1) semi-automatically detect distributed state, and (2) infer invariant over distributed state.

More concretely DInv analyzes Go programs and can:

  * Identify variables at processes that influence messaging behaviour or data sent to other processes
  * Identify data relationships between these variables (e.g., server.counter >= client.counter)


## Usage

 * Use `instrumenter.go` to replace dump annotations in your program with statements to dump state at those points as follows: ` go run instrumenter.go > ../TestPrograms/assignment1_modified.go`

 * Run the instrumented program in the usual way to generate logs (e.g., `go run assignment1_modified.go`)

 * Run the log merger to concatenate logs from 2 nodes into the format expected by Daikon: `go run LogMerger.go`

 * A file named `daikonLog.txt` will be generated in the base directory which is in the format expected by Daikon. Use this log to infer invariant with Daikon.

## Examples

The examples directory contains a number of small distributed systems
that have been instrumented with DInv's interface. Using DInv is a
mulit step procedure. Each of the example programs is paired with a
script `run.sh`. The script oversees the execution of each stage, and
pipes each steps output into its successor. The scripts have a
standard interface

    * `.\run.sh` : Standard execution, results printed to standard out,
      all generated files removed after execution.
    
    + Options :
        * -d : dirty run, the generated files at each stage are
      not remove after execution
        *   -c : cleanup generated files

### Hello DInv
Hello DInv is our introductory example program, in which a client
sends "Hello" to a server, and the server responds with the time. The execution of each
stage of DInv has been bash scripted in
`DInvDirectory/examples/helloDinv/run.sh` to run the example
 * `cd DinvDirectory/examples/helloDinv/1
 * `.\run.sh`
The inferred data invariant of this execution are the message strings
passed between client and server.

### Sum
Sum is a client server system. The client randomly generates values
for two variables `term1` and `term2` over a constant range. The terms are sent to the
server which adds them, and sends back the results as the variable
`sum`. Inferred invariant for this example include
    * `server-sum = client-term1 + client-term2`
    * `server-sum >= client-term2`
    * `server-sum >= client-term1`

### Linear
Linear is a three host system. The hosts are `client`, `coeff`, and
`linn`. The hosts only pass messages with their neighbours ie `client
<--> coeff <--> linn`.

Similar to the sum system the `client` randomly generates two terms,
packages them, and sends them to its neighbour `coeff`. `coeff`
generates a coefficient for the first term, then packages it along
with the first two terms and sends to `linn`. `linn` calculates the
linear equation `linn = coeff * term1 + term2`. The variables `sum` is
propagaed back through `coeff` to the `client` host.

detected invariant include
 * `linn > coeff`
 * `linn > term1`
 * `linn > term2`
 

