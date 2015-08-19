# Raft
Basic Raft reference implimenation using hashicorp's Raft library. This implementation was forked from [woodsaj/raft](https://github.com/woodsaj/raft).

#Modified Repositories
This implementation requries the insturmentation of the [hashicorp
raft](http://www.github.com/hashicorp/raft) repository, along with its
underlying communication mechanism
[go-msgpack](http://www.github.com/hashicorp/go-msgpack/)

#Directory structure
	* raft
		* data (raft database files generated at runtime )
			* raft8080.db
			* raft8081.db
			* ...
		* peers
			* peers.json (a list of all hosts)
		* snapshots ( history of snapshot requests by all hosts )
		* main.go ( entry point, and configuration file )
		

			