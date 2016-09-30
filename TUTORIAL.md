TODO document PackM -> makes shiviz output more usable

# Setup
Before getting started, install Go, Dinv, GoVector, and Daikon. Refer
to the README for detailed instructions. In addition, access to the
source code of the system you want to analyze is needed.

# Instrumenting Network Calls
Dinv establishes relationships between events at different hosts by
attaching Vector Clocks to messages. Each clock update is permanently
written to a log on disk.

Instrumenting message can be done in multiple ways. You should first
try to use Dinv's automatic instrumentation feature and only manually
intervene in case of failure.

## Auto instrumentation with GoVector
Dinv uses a tool called GoVector to manage Vector Clocks at runtime.
GoVector must somehow attach the clock to every outgoing message and
strip it off before the message's data is passed back to the
application. To help developers preparing their code, GoVector
provides pre-defined wrappers for common networking functions; TODO
reference the README for a full list of supported methods.

GoVector provides a command line tool to automatically find and
instrument networking calls. For instance, to instrument the sum
example, run

    GoVector -dir="$GOPATH/src/bitbucket.org/bestchai/dinv/examples/sum"

This will yield the following output:

    Source Built
    Source Built
    Wrappers Built
    var conn net.PacketConn
    var conn net.PacketConn

The last two lines reveal that two network calls have been
instrumented. GoVector has injected a new import
(`github.com/arcaneiceman/GoVector/capture`) in
`server/lib/server.go`, and wrapped `ReadFrom` and `WriteTo`:

    _, addr, err := conn.ReadFrom(buf[0:])
    [...]
	conn.WriteTo(msg, addr)
    
has been replaced by

    _, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
    [...]
	capture.WriteTo(conn.WriteTo, msg, addr)
    
The rest of the program is untouched. GoVector transparently attaches
and strips clocks without changing the program's semantics.

If GoVector doesn't work as expected, the capture functions must be
added manually. If the examined system uses not supported net
functions or has a custom en/decoding strategy, a system-specific
strategy has to be applied. 

## Manual transport of vector clocks
If GoVector failed to instrument your code, you need to manually
attach vector clocks to messages. It's helpful to take a look at
the predefined functions, in this case `capture.WriteTo`:

```go
func WriteTo(writeTo func([]byte, net.Addr) (int, error), b []byte, addr net.Addr) (int, error) {
    buf := dinvRT.Pack(b)
    n, err := writeTo(buf, addr)
    return n, err
}
```
    
First, the unmodified message buffer is passed to `dinvRT.Pack()` and
the return value assigned to `buf`. `dinvRT.Pack()` writes the current
vector clock to disk, and returns the message buffer with the vector
clock attached to it.

Following that, the modified buffer is passed to the user-specified
networking function (i.e., `conn.WriteTo`) and the result is returned.

When implementing a custom vector clock solution `dinvRT.Pack` has to
be used to initiate a clock tick and get the encoded vector clock. On
the receiving side, `dinvRT.Unpack` must be used to ingest the update.

    // sending
    vc := dinvRT.Pack(nil)
    // vc is a byte array ([]byte) containing only the vector clock
    // without additional information

    ...

    // receiving
    var dummy []byte
    dinvRT.Unpack(vc, &dummy)
    // dinv will have logged the new vector clock
    // dummy will contain no data; it's only used to satisfy the type system 

### Example: vector clock transportation using HTTP headers
Dinv does not automatically instrument HTTP communication. A common
approach is to transport the vector clock in a custom HTTP header. The
following example demonstrates how to get the current vector clock,
and insert the base64 representation into a header field named
"VectorClock" .

```go
func handleHTTP(w http.ResponseWriter, r *http.Request) {
  vc := dinvRT.Pack(nil)
  w.Header().Set("VectorClock", base64.URLEncoding.EncodeToString(vc))
  w.Write(getReponse(r))
}
```

The code handling an HTTP response is not quite as compact due to
error handling. In essence however it performs the steps backwards:
reading the encoded vector clock from the header, decode it, and pass
it to Dinv. Remember that the byte array `dummy` passed to
`dinvRT.Unpack` can be ignored.

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

### TODO Example: custom decoding
## Verifying instrumentation
After instrumenting network calls, a run of the system should yield
log files containing JSON encoded vector clocks for each process in
the current working directory. Log files are named by the schema
`${timestamp at Dinv initialization}.log-Log.txt`. Each host is
uniquely defined by the time it was started.

Each log file starts with the host initializing. Over time you can see
the host receiving messages from different hosts for the first time.
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

Look at each `*.log-Log.txt` file and check that every node included
the vector clocks from all other nodes. Often, when somethings not
working as expected you will find that only the local node's vector
clock is included.

TODO custom hostnames

# Logging variables 
Now that Dinv is able to make use of vector clocks to reason about the
order of events happing at different nodes, we can think about which
values to log at which program points. In our experience, an iterative
narrowing approach is most effective.

Pick which step to start on, depending on how familiar you are with
the system, if you already have specific invariants in mind and how
big the system is. Start by instrumenting the system in some way and
move on to a first execution and analysis before reconsidering the
selected variables.

1. logging all variables in scope at each function entry and return
2. logging all variables in scope at specific program points
3. logging specific variables at specific program points

TODO how do I invoke the first approach?

## dinvRT.Dump
Dump is one of two constructs to log variables during execution.
When a Dump statement is encountered, the variables' values are
immediately logged. Dinv analyses those values when looking
for invariants. The function signature `Dump(dumpID, variableNames
string, variableValues ...interface{})` reveals that three arguments
are expected.

The first one, `dumpID string`, must *uniquely identify* a dump
statement in a execution *across all hosts*. While no specific format
is enforced, it's usually a good idea to include a process's
hostname, port number or IP address in the dump ID. Mind that the
number of variables and their names included in a Dump statement must
stay the same over the course of an execution, e.g., they must not be
dynamic. Example:

```
Code at all nodes:
    dinvRT.Dump("node" + port + ":MemberStateDump, "var1,var2", var1.String(), var2.String())

DumpID at node listening on port 8000: node8000:MemberStateDump
DumpID at node listening on port 8001: node8001:MemberStateDump
```

From our experience it's usually a good idea to use Dump instead of
Track statements, when instrumenting a system for the first time.
Experiment with Track when the generated invariants don't match your
expectations.

## Track
Track expects the same arguments as Dump and behaves in general
similarly. The big difference is that Track doesn't immediately log
the values, but cumulates them and only *logs them at the next
send/receive* event. Only the latest value of a variable is logged.

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
dinvRT.Dump(hostname + ":MemberState", "MemberState", state.String())
```

Notice that `members` is sorted before the string is built; this is
needed to make the comparison independent of insertion order.

# Running the instrumented system
When trying out Dinv for the first time, spin up your system and let
it work for some time. If the number of nodes is dynamic, start with
2-4 nodes and let the system run between 10 and 30 seconds. You need
to make sure that the instrumented code paths are used and message are
exchanged to generate enough data that Dinv can analyze. During
execution, every process should have created and wrote the specified
variables to a log file, i.e., `1475088947.Encoded.txt`. Continue by
analyzing these files with Dinv.

Manually starting and stopping executions by hand gets tiring.
Furthermore it's desirable to make executions as deterministic as
possible to attribute changes in invariant output to instrumentation
and analysis settings. When setting up a run environment, consider the
invariants you are interested in and the causes that result in the
execution of instrumented code paths: Request keys from a
load-balanced key-value store to check for consistency or partition a
leader-election system to observe re-elections.

# Merging execution logs
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

Imagine a system with a shared counter. The nodes are arranged in a
ring and linked one way. Every node can increase the counter and
propagate the new value by passing it along to the next node. When a
node receives a value and the value is greater then its' current one,
it applies the update and forwards it; if the value is less or equal
to the current one, no action is taken.

```
state 1:           state 2:           state 3:           state 4:           state 5:
node1 ---> node2   node1 ---> node2   node1 ---> node2   node1 ---> node2   node1 ---> node2
 (1)        (1)     (2)   2    (1)     (2)        (2)     (2)        (2)     (2)        (2)
  ^          |       ^          |       ^          |       ^          |       ^          |
  |          |       |          |       |          |       | 2        |       |          |
  |          |       |          |       |        2 |       |          |       |          |
node3 <-------     node3 <-------     node3 <-------     node3 <-------     node3 <-------
 (1)                (1)                (1)                (2)                (2)
``` 

We want to check that the counter eventually converges across all
nodes. This can be expressed as `node1.counter == node2.counter ==
node3.counter`.

To allow Dinv to compare the counter between nodes, we have to log the
new value after every update: `dinvRT.Dump(nodename+".counter",
"counter", counter)`. Note, that the counter is *not logged* when a message
didn't result in the counter being updated (as in state 5).

Merging the run traces with whole cut merge is not expedient, since
there are temporal states which invalidate the invariant (state 2 and
3). Looking at the execution reveals that the invariant is never
invalidated on the granularity of a send-receive interaction. Using
send-receive merge, we gather three useful invariants:

```
p-_node1.counter_node2.counter
node1.counter == node2.counter

p-_node2.counter_node3.counter
node2.counter == node3.counter

p-_node3.counter_node1.counter
node3.counter == node1.counter
```

Formulating the first invariant, Dinv never observed a message from
node 1 to node 2, after which node2's counter wasn't at least as big
as node1's (`node1.counter <= node2.counter). Since all message
exchanges are covered, we can be assured that counter updates are
correctly dispersed.

# Running Daikon
It's time for some invariants! Assuming you are in a directory
containing dtrace files generated by Dinv, run `java daikon.daikon
*.dtrace`. Daikon prints the results to the console and additionally
saves them in files ending with ".inv.gz".

TODO: interpreting invariants
- need to be seen in context of distributed program point they are gather from
