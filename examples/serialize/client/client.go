package main

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"bitbucket.org/bestchai/dinv/instrumenter"
)

var listeningPort int
var requests = 5

func main() {
	setArgs()
	// sending UDP packet to specified address and port
	conn := setupConnection(8080, listeningPort)
	instrumenter.Initalize(fmt.Sprintf("listeningPort"))
	for i := 0; i < requests; i++ {
		msg := []int{listeningPort, i}
		instrumentedMessage := instrumenter.Pack(msg)
		_, errWrite := conn.Write(instrumentedMessage)
		printErr(errWrite)

		// Reading the response message
		var buf [1024]byte
		n, errRead := conn.Read(buf[0:])
		printErr(errRead)
		unpackedMessage := instrumenter.Unpack(buf[:n])
		typeAssertedMessage := unpackedMessage.(int)
		fmt.Printf(">>> %d\n", typeAssertedMessage)
	}
	//@dump

	os.Exit(0)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func setArgs() {
	if len(os.Args) != 2 {
		os.Exit(0)
	}
	listeningPort, _ = strconv.Atoi(os.Args[1])
	listeningPort = listeningPort + 8080
}

func setupConnection(sendingPort, listeningPort int) *net.UDPConn {
	rAddr, errR := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", sendingPort))
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", listeningPort))
	printErr(errL)

	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)
	if (errR == nil) && (errL == nil) && (errDial == nil) {
		return conn
	}
	return nil
}
