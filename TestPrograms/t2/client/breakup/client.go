package client

import (
	"fmt"
	"math/rand"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/govec"
)

const (
	SIZEOFINT     = 4
	ADDITION_ARGS = 2
	LARGEST_TERM  = 100
	RUNS          = 100
)

var (
	buf               [1024]byte
	term1, term2, sum int
)

func Client() {
	//dump
	Logger = govec.Initialize("Client", "clog.log")
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)
	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)

	for t := 0; t < RUNS; t++ {
		term1, term2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := MarshallInts([]int{term1, term2})
		// sending UDP packet to specified address and port
		_, errWrite := conn.Write(Logger.PrepareSend("sending", msg))

		//@dump
		printErr(errWrite)
		//adding local events for testing lattice /jan 23 2015
		//		for i := 0; i < 3; i++ {
		//			Logger.LogLocalEvent("Twittle Thumbs")
		//		}
		// Reading the response message

		_, errRead := conn.Read(buf[0:])
		ret := Logger.UnpackReceive("Received", buf[0:])
		printErr(errRead)

		uret := UnmarshallInts(ret)
		sum = uret[0]
		//@dump
		fmt.Printf("C: %d + %d = %d\n", term1, term2, sum)
		sum = 0
	}
	os.Exit(0)
}

var Logger *govec.GoLog
