package main

import (
	"math/rand"
	"net"

	"bitbucket.org/bestchai/dinv/examples/linear/comm"
	"bitbucket.org/bestchai/dinv/instrumenter"
)

const (
	LARGEST_COEFF = 6
)

//var debug = false

//dump
func main() {
	conn, err := net.ListenPacket("udp", ":8080")
	comm.PrintErr(err)

	//setup connection to linn server
	rAddr, errR := net.ResolveUDPAddr("udp4", ":9090")
	comm.PrintErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":8081")
	comm.PrintErr(errL)
	conn2, errDial := net.DialUDP("udp", lAddr, rAddr)
	comm.PrintErr(errDial)

	//main loop
	for {
		if err != nil {
			comm.PrintErr(err)
			continue
		}
		handleConn(conn, conn2)
		//fmt.Println("some one connected!")
	}
	conn.Close()
}

func handleConn(conn net.PacketConn, conn2 *net.UDPConn) {
	var buf [1024]byte
	var term1, term2, coeff int

	//read from client
	_, addr, err := conn.ReadFrom(buf[0:])
	args := instrumenter.Unpack(buf[0:])
	//@dump
	comm.PrintErr(err)
	//unmarshall client arguments
	uArgs := comm.UnmarshallInts(args)
	term1, term2 = uArgs[0], uArgs[1]
	coeff = rand.Int() % LARGEST_COEFF
	//marshall coefficient, with terms, send to linn server
	//if debug {
	//	fmt.Printf("Coeff: T1:%d\tT2:%d\tCoeff:%d\n", term1, term2, coeff)
	//}
	msg := comm.MarshallInts([]int{term1, term2, coeff})
	_, errWrite := conn2.Write(instrumenter.Pack(msg))
	comm.PrintErr(errWrite)
	//@dump

	//read response from linn server
	_, errRead := conn2.Read(buf[0:])
	ret := instrumenter.Unpack(buf[0:])
	//@dump
	comm.PrintErr(errRead)
	//unmarshall response from linn server
	uret := comm.UnmarshallInts(ret)
	lin := uret[0]
	//fmt.Printf("C: %d*%d + %d = %d\n", coeff, term1, term2, lin)
	//marshall response and send back to client
	msg2 := comm.MarshallInts([]int{lin})

	conn.WriteTo(instrumenter.Pack(msg2), addr)
	//@dump
}