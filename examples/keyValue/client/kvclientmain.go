// Simple client to connect to the key-value service and exercise the
// key-value RPC API (put/get/test-set).
//
// Usage: go run kvclientmain.go [ip:port]
//
// - [ip:port] : the ip and TCP port on which the KV service is
//               listening for client connections.
//
// TODOs:
// - Needs refactoring and optional support for vector-timestamps.

package main

import (
	"fmt"
	"net/rpc"
	"os"
)

// args in get(args)
type GetArgs struct {
	Key string // key to look up
}

// args in put(args)
type PutArgs struct {
	Key string // key to associate value with
	Val string // value
}

// args in testset(args)
type TestSetArgs struct {
	Key     string // key to test
	TestVal string // value to test against actual value
	NewVal  string // value to use if testval equals to actual value
}

// Reply from service for all three API calls above.
type ValReply struct {
	Val string // value; depends on the call
}

type KeyValService int

var kvService *rpc.Client

// Main server loop.
func main() {
	// parse args
	usage := fmt.Sprintf("Usage: %s ip:port\n", os.Args[0])
	if len(os.Args) != 2 {
		fmt.Printf(usage)
		os.Exit(1)
	}

	kvAddr := os.Args[1]
	// Connect to the KV-service via RPC.
	kvService, _ = rpc.Dial("tcp", kvAddr)
	//checkError(err)

	//Automatic()
	Manual()

	fmt.Println("\nMission accomplished.")
}

// If error is non-nil, print it out and halt.
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		os.Exit(1)
	}
}

func get(key string) (kvVal ValReply) {
	getArgs := GetArgs{key}
	err := kvService.Call("KeyValService.Get", getArgs, &kvVal)
	checkError(err)
	fmt.Println("KV.get(" + getArgs.Key + ") = " + kvVal.Val)
	return kvVal
}

func put(key, value string) (kvVal ValReply) {
	putArgs := PutArgs{
		Key: key,
		Val: value}
	err := kvService.Call("KeyValService.Put", putArgs, &kvVal)
	checkError(err)
	fmt.Println("KV.put(" + putArgs.Key + "," + putArgs.Val + ") = " + kvVal.Val)
	return kvVal
}

func test(key, test, value string) (kvVal ValReply) {
	tsArgs := TestSetArgs{
		Key:     key,
		TestVal: test,
		NewVal:  value}
	err := kvService.Call("KeyValService.TestSet", tsArgs, &kvVal)
	checkError(err)
	fmt.Println("KV.get(" + tsArgs.Key + "," + tsArgs.TestVal + "," + tsArgs.NewVal + ") = " + kvVal.Val)
	return kvVal

}

//Control the activity of a client by using keystrokes from the
//command line. All default values are 42
//Commands :
// g : get value
// p : put value
// t : test set
// e : exit
func Manual() {
	var input string //userInput
	for true {
		_, err := fmt.Scanf("%s", &input)
		checkError(err)
		switch input {
		case "g":
			get("my-key")
		case "p":
			put("my-key", "party-like-its-416")
		case "t":
			test("my-key", "foo", "bar")
		case "e":
			return
		default:
			usage := fmt.Sprintf("Manual Control Usage: \ng: get\np: put\nt: test\ne: exit\n")
			fmt.Println(usage)
		}
	}
}

//automated client execution
func Automatic() {
	get("my-key")
	put("my-key", "party-like-its-416")
	get("my-key")
	test("my-key", "foo", "bar")
	test("my-key", "party-like-its-416", "bar")
}
