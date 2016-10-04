# Analyzing distributed systems using Dinv
TODO include fig 1 (overview of steps) from dinv paper
TODO links in paragraphs to corresponding sections

Dinv is a tool to infer likely invariants of a system by analyzing
logs generated from the system's execution.

Dinv's approach to observe network communication and variables is to
augment a system's source code. Network calls need to be instrumented
and variables need to be recorded at specific program points.

The execution of an instrumented system results in artifacts that
describe communication paths and variable's values at specific times.
The execution environment should focus on triggering behavior that you
are interested in analyzing; often the system's execution is scripted.

Dinv analyses the execution traces to arrive at "distributed program
points"; points in times at which the state of multiple hosts is
comparable. 

Multiple occurrences of the same distributed program point are grouped
and analyzed by the invariant
detector [Daikon](http://plse.cs.washington.edu/daikon/). Daikon will
produce invariants for each distributed program point.

Eventually the user must evaluated the meaning of invariants for the
correctness of the analyzed system, while considering logging
instrumentation, the execution environment and analysis settings.

# Setup
Before getting started,
install
[Go](https://golang.org/),
[Dinv](https://bitbucket.org/bestchai/dinv),
[GoVector](https://bitbucket.org/bestchai/dinv),
and [Daikon](http://plse.cs.washington.edu/daikon/). Refer to
the [README](../README.md) for detailed instructions. You also need
access to the source of the system that you want to analyze.

# TODO smallish end-to-end example
Maybe integrate with sum readme?

# Network Instrumentation
Dinv establishes a temporal precedence relationship between program
points at different hosts using vector time. 

Vector clocks are attached to messages by wrapping network read/write
calls with functions provided by a library called GoVector. 

Identifying and instrumenting send and receive functions can either be
done automatically or manually. Automatic instrumentation does only
work for a limited set of standard network functions; it does not work
for custom decoders (i.e. protobuf). *Try the automatic approach first
and instrument not processed functions manually.*

## Automatic instrumentation using GoVector
GoVector provides both the library that Dinv uses to manage vector
clocks at runtime and a command-line tool to automatically identify
and instrument networking functions in Go code. 

Assuming that GoVector is correctly installed, the following command
will instrument the [sum example](examples/sum) provided along this
document.

```bash
$ GoVector -dir="$GOPATH/src/bitbucket.org/bestchai/dinv/examples/sum" 

Source Built
Source Built
Wrappers Built
var conn net.PacketConn
var conn net.PacketConn
```

GoVector's output reveals that two network calls have been
instrumented. GoVector has injected a new import
(`github.com/arcaneiceman/GoVector/capture`) in
`server/lib/server.go`, and wrapped `ReadFrom` and `WriteTo`:

```go
_, addr, err := conn.ReadFrom(buf[0:])
[...]
conn.WriteTo(msg, addr)
```
    
has been replaced by

```go
_, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
[...]
capture.WriteTo(conn.WriteTo, msg, addr)
```
    
The rest of the program is untouched. GoVector transparently attaches
and strips clocks without changing the program's semantics.

## Manual transport of vector clocks
If GoVector failed to instrument your code, you need to manually
attach vector clocks to messages. 

```go
vc := dinvRT.Pack(nil)
// vc is a byte array ([]byte) containing only the vector clock
// without additional information

```

The function `dinvRT.Pack(msg interface{}) []byte` performs a clock
tick and returns the encoded vector clock. `capture` functions pass a
message buffer to `dinvRT.Pack`, which returns th buffer with the vector
clock timestamp attached to it. To only retrieve the vector clock
without additional information, invoke `dinvRT.Pack` with `nil`.

```go 
var dummy []byte
dinvRT.Unpack(vc, &dummy)
// dinv will have processed the received vector clock timestamp
// dummy will contain no data; it's only used to satisfy the type system 
```

The function `dinvRT.Unpack(msg []byte, pack interface{})` processes
the vector clock timestamp included in `msg` and writes any additional
information to `pack`. When `dinvRT.Pack` was invoked with `nil`,
`dummy` will be empty.

### Application: Vector clock transportation using HTTP headers
A custom HTTP header can be used to transport vector clocks when
GoVector's pre-defined functions are not applicable. The following
two code snippets will demonstrate this approach:

```go
func handleHTTP(w http.ResponseWriter, r *http.Request) {
  vc := dinvRT.Pack(nil)
  w.Header().Set("VectorClock", base64.URLEncoding.EncodeToString(vc))
  w.Write(getReponse(r))
}
```

The current vector clock is retrieved using `dinvRT.Pack` and its' base64
representation is inserted into a "VectorClock" header field.

```go
vcHeader := header.Get("VectorClock")
if vcHeader == "" {
    return fmt.Errorf("no vector clock header attached")
}

vc, err := base64.URLEncoding.DecodeString(vcHeader)
if err != nil {
    return fmt.Errorf("error base64-decoding vcHeader: %s", err.Error())
}

var dummy []byte
dinvRT.Unpack(vc, &dummy)
```

The code handling an HTTP response is not quite as compact due to
error handling. In essence however it performs the steps backwards:
The encoded vector clock is read from the header, decoded, and passed
to `dinvRT.Unpack`. Remember that the byte array `dummy` is only
needed to satisfy Go's type system; it will be empty since `nil` has
been passed to `dinvRT.Pack`.

### TODO Application: custom decoding
## Verifying instrumentation
After instrumenting network calls, a run of the system should yield
log files containing JSON encoded vector clocks for each process in
the current working directory. Log files are named by the schema
`${timestamp at Dinv initialization}.log-Log.txt`. Each node is
uniquely defined by the time it was started.

Each log file starts with the node initializing. Over time you can see
the node receiving messages from different nodes for the first time.
 `810724201.log-log.txt` could look like this: 

```
810724201 {"810724201":1}
Initialization Complete
810724201 {"810724201":2}
Sending from 810724201
810724201 {"810724201":3}
Sending from 810724201
810724201 {"810724201":4, "814645650":2}
Received on 810724201
810724201 {"810724201":5, "811635838":5, "814645650":2}
Received on 810724201
810724201 {"810724201":6, "811635838":5, "812001880":3, "812296886":7, "814645650":2}
Received on 810724201
810724201 {"810724201":7, "811635838":5, "812001880":3, "812296886":7, "814645650":2}
Sending from 810724201
```

Look at each `*.log-Log.txt` file and check that all of them include
the vector clocks from all other nodes. If not all other nodes are
included (often it is only the node that wrote to the log file), at
least one communication channel is not properly instrumented.

TODO custom hostnames

# Logging variables 
Now that Dinv is able to make use of vector clocks to reason about the
order of events happening at different nodes, we can think about which
values to capture at which program points. Dinv analyses those values
to infer invariants. In our experience, an iterative narrowing
approach is most effective.

Pick which step to start on, depending on how familiar you are with
the system, if you already have specific invariants in mind and
understand how big the system is. Start by instrumenting the system in
some way and move on to a first execution and analysis before
reconsidering the selected variables.

TODO example of these steps for one of the programs in the repository
TODO how do I invoke the first approach?

1. logging all variables in scope at each function entry and return
2. logging all variables in scope at specific program points
3. logging specific variables at specific program points

Dinv distinguishes capturing values (recording a variable's value)
from logging values (writing values to disk). Dinv exposes two
structures that capture a variables value when encountered: **Dump**
and **Track**. Dump immediately logs after capturing a value. Track
however, cumulates the values and only *logs them at the next
send/receive* event. Read on for application examples.

*From our experience it's usually a good idea to use Dump instead of
Track statements, when instrumenting a system for the first time.
Experiment with Track when the generated invariants don't match your
expectations.*

During execution of a system with instrumented networking and logging
statements, every process writes to two files: One file (i.e.,
`1475088947.Encoded.txt`) containes the specified variables, the
other one (i.e., `810724201.log-log.txt` vector clock timestamps.

## Using Dump to simultaneously record and log variables
When a Dump statement is encountered, the variables' values are
captured and immediately logged. The function signature `Dump(dumpID,
variableNames string, variableValues ...interface{})` reveals that
three arguments are expected.

The first one, `dumpID string`, must *uniquely identify* a dump
statement in a execution *across all hosts*. While no specific format
is enforced, it's usually a good idea to include a process's
hostname, port number or IP address in the dump ID. Note that the
number of variables and their names included in a Dump statement must
stay the same over the course of an execution, e.g., they must not be
dynamic. Example:

```go
// Code at all nodes:
dinvRT.Dump("node" + port + ":MemberStateDump, "var1,var2", var1.String(), var2.String())

// DumpID at node listening on port 8000: node8000:MemberStateDump
// DumpID at node listening on port 8001: node8001:MemberStateDump
```

## Using Track to accumulate state before logging
Track expects the same arguments as Dump and behaves in general
similarly. Unlike Dump, Track does not immediately log values, but
cumulates them and only *logs them at the next send/receive* event.
Only the latest value of a variable is logged.

```go
s := "dlrow_olleh"
n := 0
dinvRT.Track(port+"Track1", "s,n", s, n)

s = "hello_world"
m := 5
dinvRT.Track(port+"Track2", "s,m", s, m)

// send or receive happens
// logged values: "s=hello_world, n=0, m=5"
```

TODO talk about option to not reset kv store?
TODO concrete example where this is useful

Track can provide a more complete view of a node’s state since
variables from multiple captures can be analyzed together. This comes
at the cost of precision, since intermediate state transitions are not
reflected during a single vector time.

## Advanced state encoding
Often a system's state is not directly represented in a form that Dinv
can handle or that Daikon can reason about, i.e., not represented as a
number or string. Imagine that we want to check that all members of a
cluster have a consistent view of the network: At each point during
execution, the list of members should be equal on all hosts. The
system might store information about members in an array `members
string[]`. To compare `members` between hosts we can log a string
representation of that array:

```go
sort.Strings(members)
var state bytes.Buffer
for _, member := range members {
    state.WriteString(member + ",") 
}
dinvRT.Dump(hostname+":MemberState", "MemberState", state.String())
```

Notice that `members` is sorted before the string is built; this is
needed to make the comparison independent of insertion order.

TODO replace the example used here with example actually present in
repo and end with invariants produced

# System execution
Even though some systems differ substantively, in our experience the
approach to instrumentation, execution and analysis revealed patterns
applicable to all systems. This section gives recommendations based on
our experiences.

When trying out Dinv for the first time, spin up your system and let
it work for some time. If the number of nodes is dynamic, start with
2-4 nodes and let the system run between 10 and 30 seconds. You need
to make sure that the instrumented code paths are used and message are
exchanged to generate enough data that Dinv can analyze. During
execution, every process should have created and wrote two log files,
one containing vector clock timestamps, the other one containing
logged variables. Continue by analyzing these files with Dinv.

Manually starting and stopping executions by hand gets tiring.
Furthermore it's desirable to make executions as deterministic as
possible to attribute changes in invariant output to instrumentation
and analysis settings. When setting up a run environment, consider the
invariants you are interested in and the causes that result in the
execution of instrumented code paths: Request keys from a
load-balanced key-value store to check for consistency or partition a
leader-election system to observe re-elections.

TODO mention lib.sh, maybe how to integrate it or how a typical run
script looks like

# Mining distributed state
After executing an instrumented system, your working directory
includes a `Log.txt` and `Encoded.txt` file for every node. Dinv can
now connect the individual host states based on the information
provided by vector time. To start the merging process, run `dinv
-logmerger *Encoded.txt *Log.txt` to pass all log files to Dinv.
Advisable options are `-shiviz` to visualize the execution
using [Shiviz](http://bestchai.bitbucket.org/shiviz/) and
`-name="fruits"` to produce more readable output.

When Dinv terminates, the directory contains multiple dtrace files,
each for a distributed program point, that can next be analyzed by
Daikon.

## Merging strategies
The decision about which points to group together is made by a
pluggable merging strategy. We identified three useful strategies:
Whole cut merge, send-receive merge and total ordering merge. TODO
Consult the README for further information about their algorithmic
implementation.

Which merging strategy to use depends on the type of invariant you're
after. When analyzing the system for the first time, stick with the
default total ordering merge strategy for now. Try one of the others,
if the output doesn't match your expectation.

**Total ordering merge** groups points that happened as the result of
a "message chain". TODO example

**Whole cut merge** is applicable when you want to find an invariant
that is not invalidated *at any point* during execution. Imagine that
each node is supposed to talk to only two specific other nodes.
Checking that property using Dinv can be done by always logging a
message's target address before sending it, and merging the logs using
the whole cut merging strategy.

**Send-receive merge** relates the program point at the sender
immediately before sending a message, with the program point at the
receiving node immediately after receiving the message. This strategy
is most useful when trying to check properties based on state updates
which are spread through the network from node to node.

The ring example provided along this document uses send-receive merge.
The processes is described in the [example's README](examples/ring/README.md).

# Detecting invariants using Daikon
It's time for some invariants! Assuming you are in a directory
containing dtrace files generated by Dinv, run `java daikon.daikon
*.dtrace`. Daikon prints the results to the console and additionally
saves them in files ending with ".inv.gz".

TODO: interpreting invariants
- need to be seen in context of distributed program point they are gather from

TODO document PackM -> makes shiviz output more usable
