package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
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
	buf := make([]byte, 16)

	// after instrumentation:
	// _, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
	_, addr, err := conn.ReadFrom(buf)
	if err != nil {
		return
	}

	//@dump

	a, _ := binary.Varint(buf[:8])
	b, _ := binary.Varint(buf[8:])

	sum := a + b

	fmt.Printf("[SERVER] %d + %d = %d\n", a, b, sum)

	msg := make([]byte, 8)
	binary.PutVarint(msg, sum)

	// after instrumentation:
	// capture.WriteTo(conn.WriteTo, msg, addr)
	conn.WriteTo(msg, addr)

	//@dump

	return nil
}
