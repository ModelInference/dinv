package chainkv

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"sync"

	"bitbucket.org/bestchai/dinv/instrumenter"
)

type KVNode struct {
	kvmap    map[string]KeyValInfo
	mapMutex *sync.Mutex
}

type KeyValInfo struct {
	Key         string
	Val         string
	Unavailable bool
}

type Message struct {
	Request     string
	Key         string
	Val         string
	Unavailable int
	err         error
}

var me KVNode
var nextNode *net.UDPAddr
var listen *net.UDPConn
var id int
var storeSize int
var last bool

var myLog *os.File

//+@# Automatic Documentation by Dovid, Generated (Sun Apr 24 13:43:44 PDT 2016)
// >>> 	 err (error), n (int), buf ([512]byte), errWrite (error), out ([]byte),
// >>> 	 m (*testing.Message), err (error)
// >>> sent on line:97 by listen
//       instrumenter.Dump("err,n,buf,errWrite,out,m.Request,m.Key,m.Val,m.Unavailable,m.err",err,n,buf,errWrite,out,m.Request,m.Key,m.Val,m.Unavailable,m.err)
// <<< 	 err (error), storeSize (int),
// <<< 	 m (*testing.Message), err (error), key (int), errWrite (error),
// <<< 	 n (int), err (error), out ([]byte), keyValInfo (*testing.KeyValInfo)
// <<< received on line:61 by listen
//       instrumenter.Dump("err,storeSize,m.Request,m.Key,m.Val,m.Unavailable,m.err,err,errWrite,n,out,keyValInfo.Key,keyValInfo.Val,keyValInfo.Unavailable" ,err,storeSize,m.Request,m.Key,m.Val,m.Unavailable,m.err,errWrite,n,out,keyValInfo.Key,keyValInfo.Val,keyValInfo.Unavailable)
//-@# End Auto Documentation
func Node(idArg, nextArg, lastArg string) {
	initNode(idArg, nextArg, lastArg)

	var buf [512]byte

	for {
		m := new(Message)
		n, _, err := listen.ReadFrom(buf[0:])

		instrumenter.Unpack(buf[:n], m)
		m.err = err
		errPrint(m.err)

		switch m.Request {
		case "PUT":
			key, err := strconv.Atoi(m.Key)

			if err != nil {
				fmt.Printf("Bad Key %s\n", m.Key)
				break
			} else if foo(id, key, last) {
				break
			} else {
				keyValInfo := new(KeyValInfo)

				keyValInfo.Key = m.Key
				keyValInfo.Val = m.Val
				keyValInfo.Unavailable = false
				me.kvmap[m.Key] = *keyValInfo
				storeSize = len(me.kvmap)
			}
			break
		case "GET":
			break
		case "DIE":
			break
		default:
			fmt.Printf("Unknown Request %s\n", m.Request)
			break
		}

		out := instrumenter.Pack(m)
		_, errWrite := listen.WriteToUDP(out, nextNode)
		errPrint(errWrite)

		if m.Request == "DIE" {
			listen.Close()
			os.Exit(1)
		}

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

	myLog, err = os.OpenFile(idMap[idArg]+".alog", os.O_WRONLY|os.O_CREATE, 0777)
	errPrint(err)
}

func nLog(message string) {
	message = message + " (" + time.Now().String() + ") "
	fmt.Println(message)
	myLog.WriteString(message + "\n")
}

func errPrint(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var idMap = map[string]string{
	"8081": "A",
	"8082": "B",
	"8083": "C",
	"8084": "D",
	"8085": "E",
}
