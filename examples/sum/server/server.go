package main

import (
	"bitbucket.org/bestchai/dinv/examples/sum/marshall"
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

func listenAndRespond(conn net.PacketConn) (err error) {
	var buf [2 * marshall.IntSize]byte

	// after instrumentation:
	// _, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
	_, addr, err := conn.ReadFrom(buf[0:])
	if err != nil {
		return
	}

	//@dump

	summands := marshall.UnmarshallInts(buf[0:])
	sum := summands[0] + summands[1]

	msg := marshall.MarshallInts([]int{sum})

	// after instrumentation:
	// capture.WriteTo(conn.WriteTo, msg, addr)
	conn.WriteTo(msg, addr)

	//@dump sending to client

	return nil
}
