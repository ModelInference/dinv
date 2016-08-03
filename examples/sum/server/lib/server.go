package server

import (
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/dinvRT"
)

const SIZEOFINT = 4

type rpcResponse struct {
	term1, term2, sum int
}

var (
	buf [1024]byte
)

func Server() {
	conn, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	printErr(err)

	//main loop
	for {
		if err != nil {
			printErr(err)
			continue
		}
		handleConn(conn)
		//fmt.Println("some one connected!")
	}
	conn.Close()
}

func handleConn(conn net.PacketConn) {
	var rpc rpcResponse
	var args []byte
	_, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
	dinvRT.Unpack(buf[0:], &args)
	printErr(err)
	//@dump
	uArgs := UnmarshallInts(args)
	rpc.term1, rpc.term2 = uArgs[0], uArgs[1]
	rpc.sum = rpc.term1 + rpc.term2
	msg := MarshallInts([]int{rpc.sum})
	capture.WriteTo(conn.WriteTo, dinvRT.Pack(msg), addr)
	//@dump sending to client
}
