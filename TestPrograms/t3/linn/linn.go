package main

import (
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/TestPrograms/t3/comm"
	"bitbucket.org/bestchai/dinv/govec"
)

//var debug = false

//dump
func main() {
	Logger = govec.Initialize("linn", "linn.log")
	conn, err := net.ListenPacket("udp", ":9090")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	comm.PrintErr(err)

	//main loop
	for {
		if err != nil {
			comm.PrintErr(err)
			continue
		}
		handleConn(conn)
		//fmt.Println("some one connected!")
	}
	conn.Close()
}

func handleConn(conn net.PacketConn) {
	var buf [1024]byte
	var term1, term2, coeff, lin int

	_, addr, err := conn.ReadFrom(buf[0:])

	//@dump
	args := Logger.UnpackReceive("Received", buf[0:])
	comm.PrintErr(err)

	uArgs := comm.UnmarshallInts(args)
	term1, term2, coeff = uArgs[0], uArgs[1], uArgs[2]
	lin = coeff*term1 + term2
	//if debug {
	//	fmt.Printf("C: %d*%d + %d = %d\n", coeff, term1, term2, lin)
	//}
	msg := comm.MarshallInts([]int{lin})

	//@dump
	conn.WriteTo(Logger.PrepareSend("Sending", msg), addr)
}

var Logger *govec.GoLog
