# Ricart-Agrawala
Ricart-Agrawala presents a mutual exclusion problem with a fully connected network topology (i.e. all nodes know of all nodes in the system). Each node requests to enter the critical section with a sequence number, typically based on a Lamport clock. Every time a nodes receives a request to enter the critical section it checks to see if it is in the critical section, if not, it replies yes and updates it's Lamport clock to be the maximum of it's current sequence number and the sequence number of the requesting node. If it is requesting the critical section, then it checks to see if the requesting sequence number is greater than it's own, if so it queues the request, if not it replies yes.

This assertion is interesting because of how simple but powerful it is, in this particular execution, the assertion must pass once first (so one node can be in the critical section) then it will fail the second time. 

## How to Run
To run the program, simply call the `./run.sh` script. It will run a basic version of the dining philosopher's code. This code does not violate the invariant, so the assertions will pass. To run it where an assertion fails, go to the code and change the line of code labeled with the comment `// CHANGE TO` in order to violate the invariant. You can re-run the code using the same script. 

To determine if the assertion has passed or failed, go to the node logs (`node*-Log.txt`) and check to see if the logs contain the message "ASSERTION FAILED". The node at which the assertion is called will have a message of "ASSERTION FAILED: \<map of values\>" or "ASSERTION PASSED: \<map of values\>". The nodes that receive the kill message from the assertion library will have a "Received ASSERTION FAILED" log message.
