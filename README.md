# DInv is a distributed system data invariant detector

Distributed system state is not easily accessible and must be pieced together from the state at the individual hosts. Developers have few tools to help them extract, check, and reason about distributed state. DInv is a suite of tools that can (1) semi-automatically detect distributed state, and (2) infer data invariants over distributed state.

More concretely, DInv analyzes Go programs and can:

  * Identify variables at processes that influence messaging behaviour or data sent to other processes:
    * host.groupLeader = self
    * node.state = election
    * client.connectionStatus = down
  * Identify data relationships between these variables:
    * server.counter >= client.counter


# Installation
-----------------------

DInv is written in [ go lang ](http://golang.org/) and requires a working installation of [Daikon ](http://plse.cs.washington.edu/daikon/).

The following instructions are for Ubuntu 14.04

The latest version of go is available through the apt package manager
    sudo apt-get install golang

(Note that DInv cannot be compiled using gccgo. Running DInv with gccgo will cause spurious errors.)

DInv is built to run within a standard go workspace environment. Go workspaces provide a mechanism for importing, and running the source code of bitbucket and github repositories. This interface is used throughout the installation, and is necessary for the instrumentation process. For detailed instructions on how to configure a go workspace  see [ how to write go code ](http://golang.org/doc/code.html#Organization)

The `GoPath` environment variable is a reference to the root of your go workspace. This variable must be set to use both the `go get` instruction, as well as our example programs.

To clone the repository run the following commands

`mkdir -p $GOPATH/src/bitbucket.org/bestchai`

`cd $GOPATH/src/bitbucket.org/bestchai`

`hg clone https://bitbucket.org/bestchai/dinv`

## Dependencies

Dinv depends on a number of remote repositories. To install them run `dependencies.sh`.
After installing the dependent software, dinv itself can be installed
by running

`go install bitbucket.org/bestchai/dinv`

**remote dependencies**

 * github.com/godoctor/godoctor/analysis/cfg
 * github.com/arcaneiceman/GoVector/govec/vclock
 * github.com/willf/bitset
 * golang.org/x/tools/go/loader
 * golang.org/x/tools/go/types
 * gopkg.in/eapache/queue.v1



## Daikon

To infer invariants on the trace files produced by DInv
[install](http://plse.cs.washington.edu/daikon/download/doc/daikon/Installing-Daikon.html#Complete-installation)  [Daikon](http://plse.cs.washington.edu/daikon/).


# Usage
-----------------------

DInv has 6 stages to its usage. 3 stages are dedicated to the
instrumentation of source code, and the final 3 running the instrumented code and mining
invariants from logs.

## Instrumentation

Variables in go code are not readily available for analysis at
runtime. In order to track variables during execution their values
must be logged. DInv users must annotate their code with `//@dump`
comments on lines of code near variables they wish to analyze.
`//@dump` statements trigger the injection of code which logs
variables in its vicinity.

Concurrency in distributed systems makes ordering events from different
hosts difficult. To keep track of time DInv piggybacks on messages and
logs when they are sent or received. Users have to instrument their messages
with the `Pack()` and `Unpack()` instructions to wrap them with
logging information.

After source code has be annotated, and communication is captured, the
code can be processed by the instrumenter. Instrumentation occurs at
the directory level. All source files within the directory will be
analyzed for annotations. The result of instrumentation is two
directories. If `foo` was the directory given to the instrumenter, the
result would be `foo` and `foo_orig`. `foo_orig` Contains the original
files, and `foo` contains instrumented code.

Running instrumented code produces the logs dinv uses to detect
invariants. Instrumented directories retain the name of the original,
preserving external dependencies. You should be able to run
instrumented code, using identical methods for uninstrumented code.

### Dump annotations (Step 1)
---------------------------------

Variable extraction is a semi-automated task. Rather than attempt to analyze the value of each variable on every line of code, 
the user must specify lines in the source code where they want to
detect invariants. To analyze the values of variables at a specific
line of code, insert the annotation `//@dump` to that line. The
`//@dump` annotation is a trigger for the instrumenter to collect
variables. The dump statements are then replaced with code that logs
the variables, their values and the time.

#### Example

```
#!go
    0   func doit( foo int) {
    1       bar := 2 * foo
    2       //@dump
    3       return bar - foo
    4   }
```

Post Instrumentation

```
#!go    
    0   func doit( foo int) {
    1       bar := 2 * foo
    2       inject.InstrumenterInit()
    3       2_vars :=interface{}{foo,bar}
    4       2_varname := []string{"foo","bar"}
    5       p_2 := inject.CreatePoint(2_vars,2_varname,"2",instrumenter.GetLogger(),instrumenter.GetStamp())
    6       inject.Encode.Encdoe(p_2)
    7       inject.ReadableLog.WriteString(p_2.String())
    8       return bar - foo
    9   }
```



### Packing and Unpacking (Step 2)
-----------------------------------

To use DInv, the library must be aware of network messages in your system.
Two methods `Pack( buffer )` and `Unpack( buffer )` must be used
prior to transmitting/receiving data. The `Pack()` function must be used on buffer
prior to sending. It adds tracking information to the buffer and logs
the sending event. `Unpack()` must be used on all received data. It
removes the tracking information added by `Pack()` and logs the
received message.

#### Usage

Add a reference to the dinv/instrumenter repository in your source
code

```
#!go
import "bitbucket.org/bestchai/dinv/instrumenter"`
```

#### Function documentation
```
#!go
func Pack(outgoingMessage []byte) []byte
```

The Pack() function prepares a network message for transit by appending
logging information to the message. Pack() also logs a local event
when it is called.

**arguments:** outgoingMessage ([]bytes) a message prior to sending

**return value:** an array of bytes containing the original message, and logging information

**postConditions:** a local event has been logged, and the logical time on the calling host has been incremented.
    
```
#!go
func Unpack(incommingMessage []byte) []byte
```

The Unpack() function removes the logging information appended by the
pack function, and logs the event locally. The return value is the
message passed to Pack()

**arguments:** incommingMessage ([]bytes) an inbound message passed over a network

**return value:** The byte array passed to Pack()
    
**preconditions:** The buffer passed to Unpack() must have been packed prior to sending. If no logging information is present in the message, an error will be thrown.

**postConditions:** a local event has been logged, and the logical time on the calling host has been incremented.

####Example

client.go
```
#!go
       0 message := "Hello World!"
       1 connection.Write( instrumenter.Pack( message )
```

server.go

```
#!go
       0 connection.Read( buffer )
       1 message := instrumenter.Unpack ( buffer )
```

This will produce a log for both the client and server

client.log-Log.txt

    client {"client":1}
    Initialization Complete
    client {"client":2}
    Sending from client.go:1

server.log-Log.txt
    
    server {"server":1}
    Initialization Complete
    server {"server":2, "client":2}
    Received on server.go:1
    

For more information on the runtime api checkout `/instrumenter/api`

### Running the Instrumenter (Step 3)
--------------------------------------

After source code has be annotated, and communication has been
instrumented to produce logs your project will be ready for instrumentation.
Instrumentation runs at the directory level. Running DInv's
instrumenter on a directory will trigger the duplication of that
directory. The result is two directories, for example running
`dinv -i someDir` will instrument all files within `someDir` and copy
all original files to `someDir_orig`. The instrumented files are placed into
the original directory to preserve external
dependencies. The un-instrumented files will be placed into
`someDir_orig`.

running `dinv -i foo`

produces:

 * `foo` instrumented directory
 * `foo_orig` directory prior to instrumentation

## Merging Logs
---------------
Dinv mines invariants by analysing the logs generated by instrumented code. Instrumented code retains external dependences and can be run like the original code. The product of running instrumented code is a set of log files for each host in the system. The logs contain vector clocks maintained over the course of the execution, along with variables, and their values at the `//@dump` annotations. The vector clocks are used to model the history of the execution. By examining the execution total orderings between host communication can be determined. Program points logged in these total orderings are merged together. The merged points are grouped together by both host and line number the points were collected on. Variable values from these groups are transcribed into Daikon readable trace files. The last step of the execution is to run Daikon on the generated trace files, to infer invariants on their values.

### Execution of instrumented code ( Step 4 )
---------------------------------------------


Instrumented code retains external dependences, and can be run the same as uninsturmented
code. The difference is insturmented code generates logs. The logs are written to the directory of the instrumented source code. There are three different styles of
logs produced and it is important to know the difference.

 * GoVector Logs - Are generated by `Pack()` & `Unpack()`. They contain a
   history of vector timestamps.
 * Program Point Log ( Encoded ) - Generated by `//@dump` annotations. They
   contain encoded variables, and vector timestamps
 * Program Point Log ( Readable ) - Generated by `//@dump`
   annotations. They contain human readable variables, and vector timestamps

####GoVector logs

GoVector logs are generated with the format
`packagename_timestampId_log.Log.txt` the logs contain a
[ShiViz](http://bestchai.bitbucket.org/shiviz/) log generated on the host with a corresponding timestamp.
These entries in these logs are generated by calls to the `Pack()` and `Unpack()`
methods. Log entries are of the form.

    SelfHostID { "hostID:time", ..., "hostID:time" }
    Event Message - filename:lineOfCode SelfHostID

####Program point variable value Logs

Program point variable value logs are generated by `//@dump`
annotations. They contain the variable names and their values, the
location in the program they were extracted from, and the time the
variables had those values.

Two logs are generated for each host of the form

 * PackageName-HostID-Encoded.txt
 * PackageName-HostID-Readable.txt

The readable log is a debugging tool and is not used again by Dinv. The log entries have the format

    hostId_package_filename_lineNumber
    {"hostid":clockvalue, ... }
    {variablename : value , ...}
    

The encoded log used by the log merger in the next step. Conceptually the encoded logs are structured the same as the readable ones.

### Merging Logs ( Step 5 )
---------------------------

Post execution independent logging files have been written by each
instrumented host. In order to detect data invariants on the logged
variables, all the logs must be analysed together. The log merging
process tracks vector clocks logged along with variables. The
vector clocks are used to build a unified history of the execution.
The flow of variables between hosts is detected, and points of
consistency are determined based on communication patterns. These
constant distributed points have their variables aggregated together
for invariant detection. The output of the logmerger is a set of
trace files uniquely identified by the consistent distributed point
they represent.

Merging logs is hands off when compared with instrumenting. In order
to merge the logs accurately all of the encoded program point logs, and
GoVector logs generated during the execution of the instrumented code
must be given to the logmerger.

the logmerger expects as a argument the list of all GoVec logs and encoded point logs written during execution. The order of the arguments does not matter, however the inclusion of each file is necessary. In the case of a missing log an error will be thrown and the merging process terminated.


as an example consider the merging of logs collected from two host
client and server

`dinv -logmerger client-clientidEncoded.txt server-serveridEncoded.txt clientid.log-Log.txt serverid.log-Log.txt`

This example is verbose, for the sake of explanation.

The simplest way to execute the logmerger, is to move all generated
logs to a single directory and run

`dinv -l *Encoded.txt *Log.txt`

### Detecting Invariant ( Step 6 )
-----------------------------------


The output of the logmerger is a set of dtrace files. The dtrace files
have names corresponding to the program points where the values they track have been
extracted from. An individual point has a name of the form

`point` = `_hostid_packageName_FileName_lineNumber` 

These points identify `//@dump` statements along with the host id.
Trace files are named after sets of points across multiple hosts which
data has flowed through. Trace files have the naming convention

point_point_...point.dtrace

daikon detects data invariants on the variables written into trace
files. To detect and print out the invariants on a trace file
`points.dtrace` run the following.

    java daikon.Daikon points.dtrace

this produces a .inv.gz file which can be printed by running

    java daikon.PrintInvariants points.inv.gz

The following bash script prints the invariant for all trace files in
a directory.

```
#!bash
    for file in ./*.dtrace; do
            java daikon.Daikon $file
    done
    for trace in ./*.gz; do
        java daikon.PrintInvariants $trace >> output.txt
    done
```
# Examples
-----------------------

The examples directory contains a number of small distributed systems
that have been instrumented with DInv's interface. Using DInv is a
multi step procedure. Each of the example programs is paired with a
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
 * `./run.sh`
The inferred data invariant of this execution are the message strings
passed between client and server.

### Sum

Sum is located in the examples/sum directory. A script `run.sh`
mannges every stage of execution. In order to run the script, use the
following command.

    DinvDirectory/examples/sum/run.sh

Sum is a client server system. The client randomly generates values
for two variables `term1` and `term2` over a constant range. The terms are sent to the
server which adds them, and sends back the results as the variable
`sum`. Inferred invariant for this example include
   * `server-sum = client-term1 + client-term2`
   * `server-sum >= client-term2`
   * `server-sum >= client-term1`

### Linear

The linear example program is located in `exampels/linear`. The
execution of the program is scripted in `run.sh`. In order to run the
script, use the following command.

    DinvDirectory/examples/linear/run.sh

Linear is a three host system. The hosts are `client`, `coeff`, and
`linn`. The hosts only pass messages with their neighbours IE `client
<--> coeff <--> linn`.

Similar to the sum system the `client` randomly generates two terms,
packages them, and sends them to its neighbour `coeff`. `coeff`
generates a coefficient for the first term, then packages it along
with the first two terms and sends to `linn`. `linn` calculates the
linear equation `linn = coeff * term1 + term2`. The variables `sum` is
propagated back through `coeff` to the `client` host.

Detected invariant include:

    linn > coeff
    linn > term1
    linn > term2