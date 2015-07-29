package main

import (
	"fmt"
	"net"
	"time"

	"github.com/wantonsolutions/GoVector/govec"
)

//@dump
func main() {
	Logger = govec.Initialize("Server", "testlog.log")
	conn, err := net.ListenPacket("udp", ":8080")
	//	if err != nil {
	//		fmt.Println(err)
	//		os.Exit(1)
	//	}
	printErr(err)

	for {
		if err != nil {
			printErr(err)
			continue
		}
		handleConn(conn)
		fmt.Println("some one connected!")
		//@dump
	}
	conn.Close()

}

func handleConn(conn net.PacketConn) {
	var buf [512]byte

	_, addr, err := conn.ReadFrom(buf[0:])
	Logger.UnpackReceive("Received", buf[0:])
	printErr(err)
	msg := fmt.Sprintf("Hello There! time now is %s \n", time.Now().String())
	conn.WriteTo(Logger.PrepareSend("Sending", []byte(msg)), addr)
	//@dump
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

var Logger *govec.GoLog
