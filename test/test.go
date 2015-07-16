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
	RUNS          = 3
)

var (
	cConn *net.UDPConn
	sConn net.PacketConn

	buf                  [1024]byte
	cTerm1, cTerm2, cSum int
	sTerm1, sTerm2, sSum int
	done                 chan int
	conn                 *toy
)

func main() {
	Init()
	go Server()
	go Client()
	<-done
	os.Exit(0)
}

func Client() {
	for t := 0; t < RUNS; t++ {
		cTerm1, cTerm2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		conn.Write()
		conn.Read()
		msg := MarshallInts([]int{cTerm1, cTerm2})
		if cTerm1 < 5 { //dummy should not be picked up
			dummy := 6
			print(dummy)
		}
		// sending UDP packet to specified address and port
		_, errWrite := cConn.Write(Logger.PrepareSend("", msg))
		//@dump
		printErr(errWrite)
		//adding local events for testing lattice /jan 23 2015
		//		for i := 0; i < 3; i++ {
		//			Logger.LogLocalEvent("Twittle Thumbs")
		//		}
		// Reading the response message

		_, errRead := cConn.Read(buf[0:])
		ret := Logger.UnpackReceive("Received", buf[0:])
		printErr(errRead)

		uret := UnmarshallInts(ret)
		cSum = uret[0]
		fmt.Printf("C: %d + %d = %d\n", cTerm1, cTerm2, cSum)
		cSum = 0
	}
	done <- 0
}

func Server() {
	for t := 0; t < RUNS; t++ {
		var buf [1024]byte
		var sTerm1, sTerm2, sSum int

		_, addr, err := sConn.ReadFrom(buf[0:])
		args := Logger.UnpackReceive("Received", buf[0:])
		printErr(err)
		uArgs := UnmarshallInts(args)
		sTerm1, sTerm2 = uArgs[0], uArgs[1]
		sSum = sTerm1 + sTerm2
		fmt.Printf("S: %d + %d = %d\n", sTerm1, sTerm2, sSum)
		msg := MarshallInts([]int{sSum})
		sConn.WriteTo(Logger.PrepareSend("Sending", msg), addr)
	}
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
	l := int(i)
	k := int(j)
	l = l + k
	print(l)
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
	l := int(i)
	k := int(j)
	l = l + k
	print(l)
	return unmarshalled
}

func Init() {
	conn = &toy{id: 5}
	Logger = govec.Initialize("self", "self.log")
	//setup receiving connection
	sConn, _ = net.ListenPacket("udp", ":8080")

	//Set up sending connection Address
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)
	cConn, _ = net.DialUDP("udp", lAddr, rAddr)

	done = make(chan int)
}

type toy struct {
	id int
}

func (t *toy) Read() {
	print(t.id)
}

func (t *toy) ReadFrom() {
	print(t.id)
}

func (t *toy) Write() {
	print(t.id)
}

func (t *toy) WriteTo() {
	print(t.id)
}

var Logger *govec.GoLog
