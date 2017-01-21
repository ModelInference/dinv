#Runtime

Dinv's runtime enviornment tracks vector timestamps, encodes and
decodes messages, logs variables, and performs distributed asserts.
Timestamps are tracked automatically. Message encoding and decoding
are handled via `pack` and `unpack` methods. Logging is implemented in
two methods `dump` and `track` which have differnt logging semantics.
Finally distributed asserts provide an expressive api for asserting
conditions on distributed variables, documentation below.

## Encoding messages
Dinv tracks vector time stamps at runtime. To implement the [vector
clock algorithm](https:en.wikipedia.org/wiki/Vector_clock), vector
clocks must be sent across the network. The `pack` and `unpack`
methods encode and decode structures, and append vector clocks to
network payloads.

Pack takes an an argument a set of bytes msg, and returns that set
of bytes with all current logging information wrapping the buffer.
This method is to be used on all data prior to communcition
PostCondition the byte array contains all current logging
information
```go
func Pack(msg interface{}) []byte 
```

PackM operates identically to Pack, but allows for custom messages
to be logged
```go
func PackM(msg interface{}, log string) []byte 
```

Unpack removes logging information from an array of bytes. The bytes
are returned with the logging info removed.
This method is to be used on all data upon receving it
Precondition, the array of bytes was packed before sending
```go
func Unpack(msg []byte, pack interface{}) 
```

UnpackM acts identically to Unpack, but allows for custom messages to be
logged along with vector clocks.
```go
func UnpackM(msg []byte, pack interface{}, log string) 
```

#Logging variables
To infer data invariants Dinv requires a log of variabls. Dinv logs
variables with two functions `dump` and `track`. These two functions
have differnt semantics, and both have their own advanges and
disadvantages. In general `dump` is more precise, but lacks
generality, while `track` is more general, but lacks precision.


Dump logs the values of variables passed in as a set of varadic
arguments. did is the dump id, it must be unique to the dump
statement, and the host. If a dump statement is constructed by hand,
the reccomended did is a line number + packagename + port number for
example 49-api.go-8080. names is a list of variable names logged
along with the values. names must be formatted as variable name
comma variable name. The comma is used to split the list and must be
included. Furthermore the number of names must correspond to the
number of values or the Dump will panic. Example:
dinvRT.Dump("49-api.go-8080","counter,variable2,buffersize",counter,variable2,buffersize)

```go
func Dump(did, names string, values ...interface{})
```

Track shares the signature and conventions of Dump, but logs to a
key value store as opposed to directly to disk. The key value store
is written to disk whenever a host increments is vector time (either
sending or receiving a message). The purpose of this functionality
is to create a 1 to 1 correspondiance between logged variables and
vector timestamps. Track statements are conceptually different than
dump statements in that they log the values of variables which may
be out of scope or non existant at the time their values are written
to disk. Track is intended to capture the summary of a hosts state
during its transition of vector time.

```go
func Track(did, names string, values ...interface{})
```


# DistributedAsserts
A distributed assert library in GoLang. This implements what we call "locally blocking time-based" asserts. These asserts are locally blocking in the sense that they block the thread calling the assert, thus it blocks the local thread. It is time-based because the assert is not run immediately. In order to ensure that the distributed snapshot taken of all the nodes contains data within a reasonable time frame, we utilize physical clocks and schedule the assert to be taken at a specific time. This allows us to reason about when the state was taken from each node and allows for a stronger assertion, i.e. the programmer knows that if the assertion fails, it failed because the state of the system from time t<sub>0</sub> to t<sub>1</sub> was a bad state. 

## Run Sample Code
To test the library, go to the test folder and run a test of your choosing. Each test will have a comment of a variable you can change which will trigger the assertion for that test. You can run it once with the assertion and once without the assertion to see the difference in behavior. Each test will have a README that describes in more detail how to run the tests and where the assertion is. The tests used are modified from [here](https:bitbucket.org/bestchai/dinv).

## Note
If running on macOS Sierra and using GoLang 1.6 this will throw a run-time fatal error. This is because GoLang 1.6 is not compatible with macOS Sierra, the programs appear to run fine though. See [here](https:github.com/golang/go/issues/17492).
*** ****

#Instructions
How to use!

##Initalization
Any node that intends to be asserted over must call:

```go
InitDistributedAssert(addr string, neighbours []string, processName string)
```



- addr: a free port that this process can recieve on.

- neighbours: a list of the ip:ports chosen by other processes to recieve on.

- processName: the name used for the log files


##Specify assertions:
Then, any variables that the node intends to expose must have the following called:

```go
AddAssertable(name string, pointer interface{}, f processFunction)
```



- name: the string associated with the variable

- pointer: a pointer to the variable's address

- f: a function which takes in the pointer, and returns a value. If this is nil, then the function is simply the identity function.


##Distributed Assert:
To make an assertion, call the following:

```go
Assert(outerFunc func(map[string]map[string]interface{})bool, requestedValues map[string][]string)
```



- requestedValues: this is a map from the ports listed in the neighbours array (from Init) to a list of variable names. The assert will go to each ip:port listed and request the variables in the array.

- outerFunc: this is a function that takes a map from ip:port to a map from variable name to variable value, and returns false if your assertion is violated (if it's in a "bad" state) 
