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

const modulo = 2


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
	err     error
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
		m.err = err
		errPrint(m.err)

		fmt.Printf("Recieved %s\n", m.Request)

		//go func (m *Message) {
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
				} else if key % modulo != id % modulo && !last {
					//not this nodes job to replicate
					break
				} else {
					keyValInfo := new(KeyValInfo)
					keyValInfo.Key = m.Key
					keyValInfo.Val = m.Val
					keyValInfo.Unavailable = false
					me.kvmap[m.Key] = *keyValInfo // execute the put
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
		//} (m)
		
		out := instrumenter.Pack(m)
		_, errWrite := listen.WriteToUDP(out,nextNode)
		errPrint(errWrite)

		if m.Request == "DIE" {
			listen.Close()
			os.Exit(1)
		}
		instrumenter.Dump("storeSize,m.Val",storeSize,m.Val)

	}

}

func initNode(idArg, next, lastArg string) {
	me = KVNode{}
	me.kvmap = make(map[string]KeyValInfo)
	me.mapMutex = &sync.Mutex{}
	instrumenter.Initalize(idMap[idArg])
	myAddr, err := net.ResolveUDPAddr("udp", ":"+idArg)
	listen, err = net.ListenUDP("udp", myAddr)
	errPrint(err)
	nextNode, err = net.ResolveUDPAddr("udp", ":"+next)
	id, err = strconv.Atoi(idArg)
	errPrint(err)
	last = (lastArg == idArg)
}

func errPrint(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var idMap = map[string]string {
		"8081": "A",
		"8082": "B",
		"8083": "C",
		"8084": "D",
		"8085": "E",
	}



