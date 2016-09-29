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

### TODO example: transport using HTTP headers
### TODO example: custom decoding
## Verifying instrumentation
After instrumenting network calls, a run of the system should yield
log files containing JSON encoded vector clocks for each process in
the current working directory. Filenames look like `1475088947.log-Log.txt`
(`${timestamp}.log-Log.txt`).

TODO custom hostnames

# Logging variables 
Now that Dinv can make use of vector clocks to reason about the order
of events happing at different host, we can think about which values
to log at which program points. In our experience, an iterative
narrowing approach is most effective.

Pick which step to start on, depending on how familiar you are with
the system, if you already have specific invariants in mind and how
big the system is. Start by instrumenting the system in some way and
move on to a first execution and analysis before reconsidering the
selected variables.

1. logging all variables in scope at each function entry and return
2. logging all variables in scope at specific program points
3. logging specific variables at specific program points

## dinvRT.Dump
`Dump` is one of two constructs to log variables during execution.
Dinv analyses those values when looking for invariants. The function
signature `Dump(dumpID, variableNames string, variableValues
...interface{})` reveals that three arguments are expected.

The first one, `dumpID string`, must uniquely identify a dump
statement in a execution across all hosts. While no specific format is
enforced, it's usually a good idea to include a processes' hostname,
port number or IP address in the dump ID. Mind that the variables
included in a Dump statement must always be the same over the course
of an execution, e.g., the must not be dynamic. Example:

    Code at all nodes:
        dinvRT.Dump("node" + port + ":MemberStateDump, staticVariableList, ...)

    DumpID at node listening on port 8000: node8000:MemberStateDump
    DumpID at node listening on port 8001: node8001:MemberStateDump

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

Notice that `members` is sorted before the string is built -- this is
needed to make the comparison independent of insertion order.

TODO how does automatic placement on each start/end of function work?

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
possible to attribute changes in invariant output to changed
instrumentation and analysis settings. When setting up a run script,
consider the invariants you are interested in and the causes that
result in the execution of instrumented code paths: Request keys from
a load-balanced key-value store to check for consistency or partition
a leader-election system to observe re-elections.

# Merging execution logs
After executing an instrumented system, your working directory
includes a `Log.txt` and `Encoded.txt` file for each node. Dinv can
now connect the individual host states based on the information
provided by vector time. The decision about which points to group
together is made by a pluggable merging strategy. We identified three
useful strategies: Whole cut merge, send-receive merge and total
ordering merge. TODO Consult the README for further information.

TODO use cases for different merging strategies

To start the merging process, move to the folder containing the log
files and run `dinv -logmerger *Encoded.txt *Log.txt`. Advisable
options are `-shiviz` to visualize the execution
using [Shiviz](http://bestchai.bitbucket.org/shiviz/) and
`-name="fruits"` to produce a more readable output. TODO explain

When Dinv terminates, the directory contains multiple dtrace files,
each for a distributed program point, that can be analyzed by Daikon.

# Running Daikon
It's time for some invariants! Assuming you are in a directory
containing dtrace files generated by Dinv, run `java daikon.daikon
*.dtrace`. Daikon prints the results to the console and additionally
saves them in files ending with ".inv.gz".

# Interpreting invariants
- depends on file (distributed program point) they are gather from
