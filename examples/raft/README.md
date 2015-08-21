# Raft
Basic Raft reference implimenation using hashicorp's Raft library. This implementation was forked from [woodsaj/raft](https://github.com/woodsaj/raft).

#Modified Repositories
This implementation requries the insturmentation of the [hashicorp/raft](http://www.github.com/hashicorp/raft) repository, along with its
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
		* run.sh scripted execution
		
# Usage
run.sh contains a script for each stage of Dinv's execution. The Script takes two command line arguments as follows

`./run.sh hosts seconds`

The script will instrument [hashicorp/raft](http://www.github.com/hashicorp/raft), then spawn `hosts` and run for `seconds`.

### example

`./run.sh 3 30` will spawn 3 raft hosts, which will execute for a total of 30 seconds before being killed by the script.

### cleanup

After executing `run.sh` a number of files will be generated in this directory. Furthermore, `hashicorp/raft` will be replaced with the instrumented version, with the original version located at `hashicorp/raft_orig` 
to remove the generated files in this directory and revert raft back to its original directory run.

`./run.sh -c`

			