package server

import (
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/govec"
)

const SIZEOFINT = 4

func Server() {
	Logger = govec.Initialize("Server", "slog.log")
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
	var buf [1024]byte
	var term1, term2, sum int

	_, addr, err := conn.ReadFrom(buf[0:])
	args := Logger.UnpackReceive("Received", buf[0:])
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
	fmt.Printf("S: %d + %d = %d\n", term1, term2, sum)
	msg := MarshallInts([]int{sum})
	conn.WriteTo(Logger.PrepareSend("Sending", msg), addr)
	//@dump
}

var Logger *govec.GoLog
