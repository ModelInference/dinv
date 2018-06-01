package main

import (
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/dinvRT"
	"bitbucket.org/bestchai/dinv/capture"
)

//var debug = false

//track
func main() {
	conn, err := net.ListenPacket("udp", ":9090")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	PrintErr(err)

	//main loop
	for {
		if err != nil {
			PrintErr(err)
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

	_, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])

	//@track
	dinvRT.Track("linn-pore", "term1,term2,coeff,lin", term1, term2, coeff, lin)
	PrintErr(err)

	uArgs := UnmarshallInts(buf)
	term1, term2, coeff = uArgs[0], uArgs[1], uArgs[2]
	lin = coeff*term1 + term2
	//if debug {
	//	fmt.Printf("C: %d*%d + %d = %d\n", coeff, term1, term2, lin)
	//}
	msg := MarshallInts([]int{lin})
	capture.WriteTo( //@track
		conn.WriteTo, msg, addr)
	dinvRT.Track("linn-pore", "term1,term2,coeff,lin", term1, term2, coeff, lin)
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
