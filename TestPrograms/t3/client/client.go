package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/TestPrograms/t3/comm"
	"bitbucket.org/bestchai/dinv/govec"
)

const (
	ADDITION_ARGS = 2
	LARGEST_TERM  = 100
	RUNS          = 10000
)

func main() {
	Logger = govec.Initialize("client", "client.log")
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	comm.PrintErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":7071")
	comm.PrintErr(errL)
	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	comm.PrintErr(errDial)

	var (
		buf               [1024]byte
		term1, term2, sum int
	)

	for t := 0; t < RUNS; t++ {
		term1, term2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := comm.MarshallInts([]int{term1, term2})
		// sending UDP packet to specified address and port
		_, errWrite := conn.Write(Logger.PrepareSend("", msg))
		//@dump
		comm.PrintErr(errWrite)

		// Reading the response message

		_, errRead := conn.Read(buf[0:])
		ret := Logger.UnpackReceive("Received", buf[0:])
		//@dump
		comm.PrintErr(errRead)

		uret := comm.UnmarshallInts(ret)
		sum = uret[0]
		fmt.Printf("C: x*%d + %d = %d\n", term1, term2, sum)
		sum = 0
	}
	os.Exit(0)
}

var Logger *govec.GoLog
