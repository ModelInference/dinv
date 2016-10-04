# ring shared counter example
This system keeps a counter that is kept contistent on all nodes. The
nodes are arranged in a ring and linked one way. Every node can
increase the counter and propagate the new value by passing it along
to the next node. When a node receives a value and the value is
greater then its' current one, it applies the update and forwards it;
if the value is less or equal to the current one, no action is taken.

This diagram visualizes an update cycle:

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
nodes. This can be expressed as `node1-counter == node2-counter ==
node3-counter`.

To allow Dinv to compare the counter between nodes, we have to log the
new value after every update: `dinvRT.Dump(nodeID, "counter",
counter)`. Note, that the counter is *not logged* when a message
didn't result in the counter being updated (as in state 5).

Merging the run traces with whole cut merge is not expedient, since
there are temporal states which invalidate the invariant (state 2 and
3). Looking at the execution reveals that the invariant is never
invalidated on the granularity of a send-receive interaction. Using
send-receive merge, we gather three useful invariants:

```
p-_[N8002]_[N8001]:::_[N8002]_[N8001]
N8002-counter == N8001-counter

p-_[N8003]_[N8002]:::_[N8003]_[N8002]
N8003-counter == N8002-counter

p-_[N8001]_[N8003]:::_[N8001]_[N8003]
N8001-counter == N8003-counter
```

Formulating the first invariant, Dinv never observed a message from
node 1 to node 2, after which node2's counter wasn't at least as big
as node1's (`N8001-counter <= N8002-counter`). Since all message
exchanges are covered, we can be assured that counter updates are
correctly dispersed.
