# DInv: distributed invariant detector

State in a distributed system is not easily accessible and must be pieced together from the state at the individual nodes. Developers have few tools to help them extract, check, and reason about distributed state.

DInv is a suite of tools to (1) semi-automatically detect distributed state, and (2) infer invariants over distributed state.

More concretely DInv analyzes Go programs and can:

  * Identify variables at processes that influence messaging behavior or data sent to other processes
  * Identify data relationships between these variables (e.g., server.counter >= client.counter)


## Usage

 * Use `instrumenter.go` to replace dump annotations in your program with statements to dump state at those points as follows: ` go run instrumenter.go > ../TestPrograms/assignment1_modified.go`

 * Run the instrumented program in the usual way to generate logs (e.g., `go run assignment1_modified.go`)

 * Run the log merger to concatenate logs from 2 nodes into the format expected by Daikon: `go run LogMerger.go`

 * A file named `daikonLog.txt` will be generated in the base directory which is in the format expected by Daikon. Use this log to infer invariants with Daikon.

## Examples

### Hello DInv
Hello DInv is our introductory example program, in which a client
sends "Hello" to a server, and the server responds with the time. The exectuion of each
stage of DInv has been bash scripted in
`DInvDirectory/examples/helloDinv/run.sh` to run the example
 * `cd DinvDirectory/examples/helloDinv/1
 * `.\run.sh`
The infered data invarents of this execution are the message strings
passed beween client and server.
