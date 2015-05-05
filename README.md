# Distributed Invariant Detector

Instructions

 * Use `instrumenter.go` to replace dump annotations in your program with statements to dump state at those points as follows: ` go run instrumenter.go > ../TestPrograms/assignment1_modified.go`

 * Run this instrumented program in the usual way to generate logs: `go run assignment1_modified.go`

 * Run the log merger to concatanate logs from 2 nodes in to the format expected by Daikon: `go run LogMerger.go`

 * A file named `daikonLog.txt` will be generated in the base directory which is in the format expected by Daikon. Use this log to infer invariants using Daikon.