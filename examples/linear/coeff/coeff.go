package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/dinvRT"
	"github.com/arcaneiceman/GoVector/capture"
)

const (
	LARGEST_COEFF = 6
)

//var debug = false

//track
func main() {
	conn, err := net.ListenPacket("udp", ":8080")
	PrintErr(err)

	//setup connection to linn server
	rAddr, errR := net.ResolveUDPAddr("udp4", ":9090")
	PrintErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":8081")
	PrintErr(errL)
	conn2, errDial := net.DialUDP("udp", lAddr, rAddr)
	PrintErr(errDial)

	//main loop
	for {
		if err != nil {
			PrintErr(err)
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
	_, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
	//@track
	PrintErr(err)
	//unmarshall client arguments
	uArgs := UnmarshallInts(buf)
	term1, term2 = uArgs[0], uArgs[1]
	coeff = rand.Int() % LARGEST_COEFF
	//marshall coefficient, with terms, send to linn server
	//if debug {
	//	fmt.Printf("Coeff: T1:%d\tT2:%d\tCoeff:%d\n", term1, term2, coeff)
	//}
	msg := MarshallInts([]int{term1, term2, coeff})
	_, errWrite := capture.Write(conn2.Write, msg)
	PrintErr(errWrite)
	//@track
	dinvRT.Track("Coe-pre", "term1,term2,coeff", term1, term2, coeff)

	//read response from linn server
	_, errRead := capture.Read(conn2.Read, buf[0:])
	//@track
	PrintErr(errRead)
	//unmarshall response from linn server
	uret := UnmarshallInts(buf)
	lin := uret[0]
	dinvRT.Track("Coe-mid", "term1,term2,lin", term1, term2, lin)
	//fmt.Printf("C: %d*%d + %d = %d\n", coeff, term1, term2, lin)
	//marshall response and send back to client
	msg2 := MarshallInts([]int{lin})
	capture.WriteTo(conn.WriteTo, msg2, addr)
	//@track
	dinvRT.Track("Coe-post", "term1,term2,lin", term1, term2, lin)
}

const (
	SIZEOFINT = 4
)

func PrintErr(err error) {
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

func UnmarshallInts(args [1024]byte) []int {
	var i, j uint
	unmarshalled := make([]int, len(args)/SIZEOFINT, len(args)/SIZEOFINT)
	for j = 0; int(j) < len(args)/SIZEOFINT; j++ {
		for i = 0; i < SIZEOFINT; i++ {
			unmarshalled[j] += int(args[SIZEOFINT*(j+1)-1-i] << (i * 8))
		}
	}
	return unmarshalled
}
