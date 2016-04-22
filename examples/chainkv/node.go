package chainkv

import (
	"fmt"
	"net"
	"strconv"
	"os"
//	"time"

	"sync"

	"bitbucket.org/bestchai/dinv/instrumenter"
)


type KVNode struct {
	// Map implementing the key-value store.
	kvmap map[string]KeyValInfo
	// Mutex for accessing kvmap from different goroutines safely.
	mapMutex     *sync.Mutex
}

type KeyValInfo struct {
	Key         string
	Val         string
	Unavailable bool
}

type Message struct {
	Request string
	Key 	string
	Val 	string
	Unavailable int
}


var me KVNode
var nextNode    *net.UDPAddr 		 //Next node in the chain
var listen      *net.UDPConn         //Listening Port
var id 			int
var storeSize 	int
var last        bool

func Node(idArg, nextArg, lastArg string) {
	initNode(idArg, nextArg, lastArg)	
	var buf [512]byte

	for {
		m := new(Message)
		n, _, err := listen.ReadFrom(buf[0:])

		instrumenter.Unpack(buf[:n], m)
		printErr(err)

		fmt.Printf("Recieved %s\n", m.Request)
		go HandelMessage(m)
		
		out := instrumenter.Pack(m)
		_, errWrite := listen.WriteToUDP(out,nextNode)
		printErr(errWrite)

		if m.Request == "DIE" {
			listen.Close()
			os.Exit(1)
		}
		instrumenter.Dump("storeSize,m.Val",storeSize,m.Val)

	}

}


func HandelMessage (m *Message) {
	
	switch m.Request{
	case "PUT":
		// Acquire mutex for exclusive access to kvmap.
		//me.mapMutex.Lock()
		// Defer mutex unlock to (any) function exit.
		//defer me.mapMutex.Unlock()
		
		key, err  := strconv.Atoi(m.Key)
		if err != nil {
			fmt.Printf("Bad Key %s\n",m.Key)
			break
		} else if key % mod != id % mod && !last {
			//not this nodes job to replicate
			break
		} else {
			keyValInfo := KeyValInfo{m.Key, m.Val, true}
			me.kvmap[m.Key] = keyValInfo // execute the put
			storeSize = len(me.kvmap)
			fmt.Printf("Put(%s, %s)\n", m.Key, m.Val)
		}
		break
	case "GET":
		break
	case "DIE":
			break
	default:
		fmt.Printf("Unknown Request %s\n",m.Request)
		break
	}
}


func initNode(idArg, next, lastArg string) {
	me = KVNode{}
	me.kvmap = make(map[string]KeyValInfo)
	me.mapMutex = &sync.Mutex{}
	instrumenter.Initalize("node_"+idArg)
	myAddr, err := net.ResolveUDPAddr("udp", ":"+idArg)
	listen, err = net.ListenUDP("udp", myAddr)
	printErr(err)
	nextNode, err = net.ResolveUDPAddr("udp", ":"+next)
	id, err = strconv.Atoi(idArg)
	printErr(err)
	last = (lastArg == idArg)
}

