package chainkv

import (
	"fmt"
	"net"
	"os"
	"time"
	"bitbucket.org/bestchai/dinv/instrumenter"
)

const MESSAGES  = 11
const mod = 2

func Client(myPort, headPort, tailPort string) {
	instrumenter.Initalize("Client")
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
		m := new(Message)
		m.Request = "PUT"

		m.Key = fmt.Sprintf("%d",i)
		if i%mod ==0  {
			m.Val = "willy"
		} else {
			m.Val = "wonka"
		}
		instrumenter.Dump("m.Val",m.Val)

		out := instrumenter.Pack(m)
		_, errWrite := listen.WriteToUDP(out,head)
		printErr(errWrite)

		r := new(Message)
		n, _, err := listen.ReadFrom(buf[0:])
		printErr(err)
		instrumenter.Unpack(buf[:n], r)
		time.Sleep(1)
	}

	m := new(Message)
	m.Request = "DIE"
	out := instrumenter.Pack(m)
	_, errWrite := listen.WriteToUDP(out,head)
	printErr(errWrite)
	listen.Close()


}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
