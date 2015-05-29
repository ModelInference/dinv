package main

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
	RUNS          = 1000
)

func main() {
	//dump
	Logger = govec.Initialize("Client", "testclient.log")
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)
	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)

	var (
		buf               [1024]byte
		term1, term2, sum int
	)

	for t := 0; t < RUNS; t++ {
		term1, term2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := MarshallInts([]int{term1, term2})
		// sending UDP packet to specified address and port
		_, errWrite := conn.Write(Logger.PrepareSend("", msg))

		//@dump
		printErr(errWrite)

		// Reading the response message

		_, errRead := conn.Read(buf[0:])
		ret := Logger.UnpackReceive("Received", buf[0:])
		printErr(errRead)

		uret := UnmarshallInts(ret)
		sum = uret[0]
		fmt.Printf("C: %d + %d = %d\n", term1, term2, sum)
		//@dump
		sum = 0
	}
	os.Exit(0)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func MarshallInts(args []int) []byte {
	var i, j uint
	marshalled := make([]byte, len(args)*SIZEOFINT, len(args)*SIZEOFINT)
	for j = 0; int(j) < len(args); j++ {
		for i = 0; i < SIZEOFINT; i++ {
			marshalled[(j*SIZEOFINT)+i] = byte(args[j] >> ((SIZEOFINT - 1 - i) * 8))
		}
	}
	return marshalled
}

func UnmarshallInts(args []byte) []int {
	var i, j uint
	unmarshalled := make([]int, len(args)/SIZEOFINT, len(args)/SIZEOFINT)
	for j = 0; int(j) < len(args)/SIZEOFINT; j++ {
		for i = 0; i < SIZEOFINT; i++ {
			unmarshalled[j] += int(args[SIZEOFINT*(j+1)-1-i] << (i * 8))
		}
	}
	return unmarshalled
}

var Logger *govec.GoLog
