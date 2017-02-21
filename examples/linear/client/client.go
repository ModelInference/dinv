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
	ADDITION_ARGS = 2
	LARGEST_TERM  = 100
	RUNS          = 50
)

var debug = false

func main() {
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	PrintErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":7071")
	PrintErr(errL)
	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	PrintErr(errDial)

	var (
		buf               [1024]byte
		term1, term2, sum int
	)
	fmt.Println()
	for t := 0; t <= RUNS; t++ {
		//@track
		fmt.Printf("\rExecuting[%2.0f]", float32(t)/float32(RUNS)*100)
		term1, term2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := MarshallInts([]int{term1, term2})
		// sending UDP packet to specified address and port
		_, errWrite := capture.Write(conn.Write, msg)

		PrintErr(errWrite)

		//@track
		dinvRT.Track("C-pre", "term1,term2,sum", term1, term2, sum)
		// Reading the response message

		_, errRead := capture.Read(conn.Read, buf[0:])
		dinvRT.Track("C-post", "term1,term2,sum", term1, term2, sum)
		//@track
		PrintErr(errRead)

		uret := UnmarshallInts(buf)
		sum = uret[0]
		//if debug {
		//	fmt.Printf("C: x*%d + %d = %d\n", term1, term2, sum)
		//}
		term1 = sum
	}
	fmt.Println()
	os.Exit(0)
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
