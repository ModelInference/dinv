package chainkv

import (
	"fmt"
	"net"
	"os"
	"time"
	"github.com/arcaneiceman/GoVector/govec"
)

const MESSAGES  = 10

func Client(myPort, headPort, tailPort string) {
	Logger := govec.Initialize("client", "clientlogfile")
	// sending UDP packet to specified address and port

	myAddr, err := net.ResolveUDPAddr("udp", ":"+myPort)
	listen, err := net.ListenUDP("udp", myAddr)
	printErr(err)
	head, err := net.ResolveUDPAddr("udp", ":"+headPort)
	printErr(err)
	tail, err := net.ResolveUDPAddr("udp", ":"+tailPort)
	print(tail)
	time.Sleep(1000000)
	printErr(err)
	var buf [512]byte


	for i := 0; i < MESSAGES; i++ {
		outgoingMessage := i
		outBuf := Logger.PrepareSend("Head Node", outgoingMessage)
		_, errWrite := listen.WriteToUDP(outBuf,head)
		printErr(errWrite)

		_, _, err := listen.ReadFrom(buf[0:])
		printErr(err)
		var incommingMessage int
		Logger.UnpackReceive("Received Message From TailNode", buf[0:], &incommingMessage)
		time.Sleep(1)
	}
	listen.Close()

}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
