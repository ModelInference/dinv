package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/dinvRT"

	"github.com/arcaneiceman/GoVector/capture"
)

const addr = ":9090"

func main() {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		fmt.Printf("[SERVER] %s\n", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("[SERVER] listening on %s\n", addr)

	//main loop
	for {
		err = listenAndRespond(conn)
		if err != nil {
			fmt.Printf("[SERVER] %s\n", err.Error())
		}
	}
}

// expects messages with two 64 bit integers (2 * 8 bytes)
func listenAndRespond(conn net.PacketConn) (err error) {
	buf := make([]byte, 256)

	// after instrumentation:
	_, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
	// _, addr, err := conn.ReadFrom(buf)
	if err != nil {
		return
	}

	//@dump

	a, _ := binary.Varint(buf[:8])
	b, _ := binary.Varint(buf[8:])

	sum := a + b
	/*
		fmt.Println(buf)
		fmt.Println(buf[:8], a, readA)
		fmt.Println(buf[8:], b, readB)
	*/

	fmt.Printf("[SERVER] %d + %d = %d\n", a, b, sum)

	msg := make([]byte, 256)
	binary.PutVarint(msg, sum)

	//fmt.Println(putN, msg)

	// after instrumentation:
	capture.WriteTo(conn.WriteTo, msg, addr)
	// conn.WriteTo(msg, addr)

	dinvRT.Dump("s64", "sum,a,b", sum, a, b)

	return nil
}
