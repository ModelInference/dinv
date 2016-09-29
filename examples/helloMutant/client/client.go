package main

import (
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/instrumenter"
)

func main() {
	// sending UDP packet to specified address and port
	conn := setupConnection(8080, 18585)
	msg := "Hello DInv!"
	instrumentedMessage := instrumenter.Pack(msg)
	_, errWrite := conn.Write(instrumentedMessage)
	printErr(errWrite)

	// Reading the response message
	var (
		buf        [1024]byte
		recMessage string
	)
	n, errRead := conn.Read(buf[0:])
	printErr(errRead)
	instrumenter.Unpack(buf[:n], &recMessage)
	//typeAssertedMessage := unpackedMessage.(string)
	fmt.Println(">>>" + recMessage)
	instrumenter.Dump("n,msg", n, msg)
	os.Exit(0)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
