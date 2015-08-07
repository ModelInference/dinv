# DInv: distributed invariant detector

State in a distributed system is not easily accessible and must be pieced together from the state at the individual nodes. Developers have few tools to help them extract, check, and reason about distributed state.

DInv is a suite of tools to (1) semi-automatically detect distributed state, and (2) infer invariant over distributed state.

More concretely DInv analyzes Go programs and can:

  * Identify variables at processes that influence messaging behaviour or data sent to other processes
  * Identify data relationships between these variables (e.g., server.counter >= client.counter)


## Installation

Installing DInv is a mulit step procedure, due to dependencies on [
Daikon ](http://plse.cs.washington.edu/daikon/) and a standards
configured [ go tool ](http://golang.org/doc/code.html#Organization) 

### Installing Go Tool

#### Ubuntu

DInv is written in [ go ](http://golang.org/) and is structured around
the standard go-tool workspace structure. For a fresh install of go
checkout the [go installation guide]()

The latest version of go can be installed by running.
`sudo apt-get install golang`

**Important** DInv is built around the go compiler built into the go
tool, not gccgo. Running DInv with gccgo will cause spurious errors.

if you do not have a go workspace set yours up according to the go
standards [ how to write go code ](http://golang.org/doc/code.html#Organization)

make sure to export the `GoPath` enviornment variable

Dinv is uses mercurial as version control, if it is uninstalled run
`sudo apt-get install mercurial`

to clone the repository run the following commands
`mkdir -p $GOPATH/src/bitbucket.org/bestchai`
`cd $GOPATH/src/bitbucket.org/bestchai`
`hg clone https://bitbucket.org/bestchai/dinv`

##### Dependencies
DInv is dependent on a number of repository they can be installed as
follows

`go get github.com/godoctor/godoctor/analysis/cfg`
`go get github.com/arcaneiceman/GoVector/govec/vclock`
`go get github.com/willf/bitset`
`go get golang.org/x/tools/go/loader`
`go get golang.org/x/tools/go/types`
`go get gopkg.in/eapache/queue.v1`


install dinv

`go install bitbucket.org/bestchai/dinv`

this process is scripted in dependencies.sh

#### Daikon
In order to infer invarients on the trace files produced by DInv
complete installation of [Daikon](http://plse.cs.washington.edu/daikon/) is needed. The [complete installation instructions](http://plse.cs.washington.edu/daikon/download/doc/daikon/Installing-Daikon.html#Complete-installation) are avialable online.



## Usage
DInv has many stages to its execution


### Instrumentation
it must first be modified to track communication, and annotated in
areas where analysis should be preformed. DInv's instrumentation API
consists of two interfaces. A runtime library housed in
`instrumenter/api` provides a set of functions for analyzing network
traffic. The second interface is a set of commented annotations used
to trigger source code analysis.

#### Runtime API

For DInv to analyze network traffic, it must be made privy
to all communication. Two methods `Pack( buffer )` and `Unpack( buffer )` must be used
on transmitted data. The `Pack()` function must be used on buffer
prior to sending. It adds tracking information to the buffer and logs
the sending event. `Unpack()` must be used on all received data. It
removes the tracking information added by `Pack()` and logs the
receiving event.

As an cursory example consider the following code snippet involving two hosts
sending a message to one another. For more complete examples see the
examples library

`client.go`
    message := "Hello World!"
    connection.Write( instrumenter.Pack( message )

`server.go`
    connection.Read( buffer )
    message := instrumenter.Unpack ( buffer )

For more information on the runtime api checkout `/instrumenter/api`

#### Static Analysis API

Variable extraction for invariant detection
is semi-automated task. Rather than attempt analyze every variable on every line of code. it is left to
the user to specify areas in the source code where invariant should
be detected. In order analyze the values of variables at a specific
line of code, insert the annotation `//@dump` to that line. The
`\\@dump` annotation is a trigger for the instrumenter to collect
variables.

#### Running the Instrumenter

After capturing your network traffic, and annotating interesting
areas of your code your project will be ready for instrumentation.
Instrumentation runs at the directory level. Running DInv's
instrumenter on a directory will trigger the duplication of that
directory. The result is two directories, for example running
`dinv -i someDir` will insturment all files within `someDir` and copy
all original files to `someDir_orig`. The instrumented files put in
the original directories name in order to preserve external
dependencies. The un-instrumented files will be placed in
`someDir_orig`, which can be renamed to give the program its
uninstrmented behaviour.

running:
`dinv -i someDir`

produces:
`someDir` instrumented directory
`someDir_orig` directory prior to instrumentation

### Execution

Instrumented code can be executed exactly the same as uninstrumented
code. The difference is the generation of a number of logging files in
the directories of the running code. There are three different styles of
logs and it is important to know the difference.

####GoVector logs

GoVector logs are generated with the format
`packagename_timestampId_log.Log.txt` the logs contain a human, and
[ShiViz](http://bestchai.bitbucket.org/shiviz/) readable vector
timestamp log generated on the host with a corresponding timestamp.
These logs are generated by calls to the `Pack()` and `Unpack()`
methods, and contain logs of the form

SelfHostID { "hostID:time", ..., "hostID:time" }
Event Message - filename:lineOfCode SelfHostID

####Program point variable value Logs

Program point variable value logs are generated by `//@dump`
statements. They contain the variable names and their values, the
location in the program they were extracted from, and the time the
varibables had those values.

Two logs are generated for each host of the form

`PackageName-HostID-Encoded.txt`
`PackageName-HostID-Readable.txt`

The Readable log is a debugging tool and is not used again by Dinv.
The encoded log used by the log merger in the next step.

### Merging Logs

Merging logs is hands off when compared with instrumenting. In order
to merge the logs accurately all of the encoded program point logs, and
GoVector logs generated during the execution of the instrumented code
must be given to the LogMerger.

The LogMerger expects two equal length lists of log files as
arguments. The first list being the encoded program points, and the
second the GoVec log files. The lists should be ordered so that the
i'th point log and the i'th GoVec log correspond to the same host.

as an example consider the merging of logs collected from two host
client and server

`dinv -logmerger client-clientidEncoded.txt server-serveridEncoded.txt
clientid.log-Log.txt serverid.log-Log.txt`

This example is verbose, for the sake of explanation.

The simplest way to execute the logmerger, is to move all generated
logs to a single directory and run

`dinv -l *Encoded.txt *Log.txt`

#### Merging output

The output of the logmerger is a set of dtrace files. The dtrace files
have names corresponding to where the values they track have been
extracted from. An individual point has a name of the form

`point` = `_hostid_packageName_FileName_lineNumber` 

These points identify `//@dump` statements along with the host id.
Trace files are named after sets of points across multiple hosts which
data has flowed through. Trace files have the naming convention

point_point_...point.dtrace

dtrace files can be fed into daikon to detect invariant.


## Examples

The examples directory contains a number of small distributed systems
that have been instrumented with DInv's interface. Using DInv is a
mulit step procedure. Each of the example programs is paired with a
script `run.sh`. The script oversees the execution of each stage, and
pipes each steps output into its successor. The scripts have a
standard interface

    * `./run.sh` : Standard execution, results printed to standard out,
      all generated files removed after execution.
    
    + Options :
        * -d : dirty run, the generated files at each stage are
      not remove after execution
        * -c : cleanup generated files

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
 

