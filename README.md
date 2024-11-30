# Fly.io Distributed System Challenges [credits: gAbelli]

These are my solutions to the distributed system challenges by Fly.io, available [here](https://fly.io/dist-sys/).
All the programs are written in Go and have been tested against the [maelstrom](https://github.com/jepsen-io/maelstrom) workloads.

## Challenge 1: Echo

This is just a test program, nothing particular to say.

## Challenge 2: Unique ID Generation

In a distributed system, there are many possible strategies to assign unique IDs. Some of them are easier to implement but don't actually provide 100% certainty of never generating a duplicate ID (for example you could just generate a long enough random string and hope that it is unique), while others are safer.
As far as I know, one of the most successful algorithms is Twitter's [Snowflake](https://developer.twitter.com/en/docs/twitter-ids), which guarantees uniqueness of the IDs and also makes sure that an alphabetic sorting of the IDs will correspond almost exactly to a chronological sorting.

In this case, every server has a unique ID, and we are also under the assumption that the servers are never shut down (even though there might be network partitions). Hence, a simple approach is to have each server generate IDs incrementally (starting from 0 and growing), and prefixing this number with the server ID. This is enough to pass the tests, but again I would probably use a Snowflake-style algorithm in a real-world system.

## Challenge 3: Broadcast

### 3a: Single-Node Broadcast

This is a very simple exercise because we can just store all messages in memory. Some important considerations are the following:

- Since we don't care about the order of the messages, and the messages are guaranteed to be unique, we can store them in a set instead of an array.
- Since the RPCs can happen concurrently, we should protect this set with a lock or some other form of concurrent-safe access to the data structure. In Go it's very idiomatic to use channels, but in this case having a simple lock is probably the most straight-forward approach.

### 3b: Multi-Node Broadcast

This one is much more interesting. The idea is that we have to broadcast messages between the nodes using a gossip algorithm, but we are free to decide how to do it and we have no particular restrictions.
Obviously, one would implement this system differently depending on the use case. For example, you could desire to optimize

- the total number of information exchanges between the nodes
- the latency between the moment in which the first node receives the message and the moment in which all the nodes have seen it
- the load on each specific node

For this first version of the algorithm, we will use the topology given to us by maelstrom, so we won't have control over the load on each individual node. Since we are not required to have all messages being propagated instantly to all nodes, we can propagate messages every N milliseconds, where N is a parameter that we can tune as we wish, instead of sending an RPC for every new message received by a node. This choice increases the latency, but drastically reduces the total number of messages exchanged.
Also, an easy optimization is to keep track of the messages that have been propagated (and acknowledged) by every other node that our server can communicate with, and avoid sending them the same messages more than once.

### 3c: Fault Tolerant Broadcast

Our solution from the previous exercise is already fault tolerant because messages are re-sent if they are not acknowledged.