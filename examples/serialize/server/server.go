package main

import (
	"fmt"
	"net"
	"time"

	"bitbucket.org/bestchai/dinv/instrumenter"
)

var threads = 3
var requests int
var alive chan bool
var serialize chan bool

func main() {
	requests = 0
	conn, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		printErr(err)
		panic(err)
	}
	serialize = make(chan bool, 1)
	alive = make(chan bool, threads)
	serialize <- true
	for i := 0; i < threads; i++ {
		alive <- true
		go requestHandler(conn, i)
		//@dump
	}
	for i := 0; i < threads; i++ {
		time.Sleep(10000)
		<-alive
	}
	conn.Close()

}

func requestHandler(conn net.PacketConn, tid int) {
	<-alive
	fmt.Printf("alive %d\n", tid)
	var buf [512]byte
	for true {
		_, addr, err := conn.ReadFrom(buf[0:])
		if err != nil {
			panic(err)
		}
		<-serialize
		req := instrumenter.Unpack(buf[0:]).([]int)
		fmt.Printf("%d serving [%d:%d]\n", tid, req[0], req[1])
		requests++
		conn.WriteTo(instrumenter.Pack(requests), addr)
		serialize <- true
	}
	alive <- true
	//@dump
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
