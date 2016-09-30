# sum example
This is an example of a client server system. The client sends two
random integers to the server. The server adds them up and responds
with the sum.

`run.sh` manages the instrumentation and execution of the system, as
well as merging their generated logs, and letting Daikon analyze their
generated trace files.

# Files
* client/client.go: generates terms, sends them to the server and listens for responses
* server/server.go: adds terms, responds with sum
* run.sh: scripted execution of client and server, including instrumentation and log analysis

# Invariant
The detected data invariants should include 
`term1 + term2 = sum`
`term1 <= sum`
`term2 <= sum`
