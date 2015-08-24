package server

import (
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/instrumenter"
)

const SIZEOFINT = 4

var (
	buf               [1024]byte
	term1, term2, sum int
)

func Server() {
	conn, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	printErr(err)

	//main loop
	for {
		if err != nil {
			printErr(err)
			continue
		}
		handleConn(conn)
		//fmt.Println("some one connected!")
	}
	conn.Close()
}

func handleConn(conn net.PacketConn) {

	_, addr, err := conn.ReadFrom(buf[0:])
	args := instrumenter.Unpack(buf[0:]).([]byte)
	printErr(err)
	//@dump
	//fmt.Printf("recieved: %s of size %d, with args %d", buf, n, args)

	//adding local events for testing lattice /jan 23 2015
	//	for i := 0; i < 3; i++ {
	//		Logger.LogLocalEvent("Twittle Thumbs")
	//	}
	uArgs := UnmarshallInts(args)
	term1, term2 = uArgs[0], uArgs[1]
	sum = term1 + term2
	msg := MarshallInts([]int{sum})
	conn.WriteTo(instrumenter.Pack(msg), addr)
	//@dump sending to client
}
