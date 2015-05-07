package main

import (
	"fmt"
	"net"
	"os"

	"../govec"
)

func main() {
	Logger = govec.Initialize("Client", "testclient.log")

	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)

	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)

	// sending UDP packet to specified address and port
	msg := "get me the message !"
	_, errWrite := conn.Write(Logger.PrepareSend("Asking time", []byte(msg)))
	printErr(errWrite)

	// Reading the response message
	var buf [1024]byte
	n, errRead := conn.Read(buf[0:])
	printErr(errRead)
	incoming_msg := string(Logger.UnpackReceive("Received", buf[:n]))
	fmt.Println(">>>" + incoming_msg)
	//@dump

	os.Exit(0)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var Logger *govec.GoLog
