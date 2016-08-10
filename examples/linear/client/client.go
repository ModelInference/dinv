package main

import (
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"bitbucket.org/bestchai/dinv/dinvRT"
	"math/rand"
	"net"
	"os"
)

const (
	ADDITION_ARGS	= 2
	LARGEST_TERM	= 100
	RUNS		= 50
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
		buf			[1024]byte
		term1, term2, sum	int
	)
	fmt.Println()
	for t := 0; t <= RUNS; t++ {
		fmt.Printf("\rExecuting[%2.0f]", float32(t)/float32(RUNS)*100)
		term1, term2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := MarshallInts([]int{term1, term2})
		// sending UDP packet to specified address and port
		_, errWrite := capture.Write(conn.Write,msg)
		dinvRT.Dump("main_client_38_SIZEOFINT,main_client_38_LARGEST_TERM,main_client_38_RUNS,main_client_38_debug,main_client_38_ADDITION_ARGS,main_client_38_msg,main_client_38_errWrite,main_client_38_rAddr,main_client_38_errR,main_client_38_lAddr,main_client_38_errL,main_client_38_conn,main_client_38_errDial,main_client_38_buf", SIZEOFINT, LARGEST_TERM, RUNS, debug, ADDITION_ARGS, msg, errWrite, rAddr, errR, lAddr, errL, conn, errDial, buf)
		PrintErr(errWrite)

		// Reading the response message

		_, errRead := capture.Read(conn.Read,buf[0:])
		dinvRT.Dump("main_client_44_ADDITION_ARGS,main_client_44_LARGEST_TERM,main_client_44_RUNS,main_client_44_debug,main_client_44_SIZEOFINT,main_client_44_msg,main_client_44_errWrite,main_client_44_errRead,main_client_44_rAddr,main_client_44_errR,main_client_44_lAddr,main_client_44_errL,main_client_44_conn,main_client_44_errDial,main_client_44_buf", ADDITION_ARGS, LARGEST_TERM, RUNS, debug, SIZEOFINT, msg, errWrite, errRead, rAddr, errR, lAddr, errL, conn, errDial, buf)
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
