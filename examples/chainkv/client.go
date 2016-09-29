package chainkv

import (
	"bitbucket.org/bestchai/dinv/instrumenter"
	"fmt"
	"net"
	"os"
	"time"
)

type Cmessage struct {
	Request     string
	Key         string
	Val         string
	Unavailable int
	err         error
}

var conn *net.UDPConn
var head *net.UDPAddr
var tail *net.UDPAddr

var clientLog *os.File

func Client(myPort, headPort, tailPort string) {
	instrumenter.Initalize("Client")
	initializeClient(myPort, headPort, tailPort)

	var buf [512]byte
	for i := 0; i < len(messages); i++ {

		m := new(Cmessage)

		m.Request = "PUT"
		m.Key = fmt.Sprintf("%d", i)
		m.Val = messages[i]
		out := instrumenter.Pack(m)

		cLog(fmt.Sprintf("m.Key = %s", m.Key))

		_, errWrite := conn.WriteToUDP(out, head)
		printErr(errWrite)

		r := new(Cmessage)
		n, _, err := conn.ReadFrom(buf[0:])
		printErr(err)
		instrumenter.Unpack(buf[:n], r)
		time.Sleep(1)

	}

	shutdown()

}

func initializeClient(myPort, headPort, tailPort string) {
	myAddr, err := net.ResolveUDPAddr("udp", ":"+myPort)
	conn, err = net.ListenUDP("udp", myAddr)
	printErr(err)
	head, err = net.ResolveUDPAddr("udp", ":"+headPort)
	printErr(err)
	tail, err = net.ResolveUDPAddr("udp", ":"+tailPort)
	clientLog, err = os.OpenFile("Client.alog", os.O_WRONLY|os.O_CREATE, 0777)
	printErr(err)
	time.Sleep(1000000)
	printErr(err)
}

func shutdown() {
	m := new(Cmessage)
	m.Request = "DIE"
	out := instrumenter.Pack(m)
	_, errWrite := conn.WriteToUDP(out, head)
	printErr(errWrite)
	conn.Close()
}

func cLog(message string) {
	message = message + " (" + time.Now().String() + ") "
	fmt.Println(message)
	clientLog.WriteString(message + "\n")
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var messages = []string{
	"It", "was", "the", "best", "of", "times,", "it", "was", "the", "worst", "of", "times,", "it", "was", "the", "age", "of", "wisdom,", "it", "was", "the", "age", "of", "foolishness,", "it", "was", "the", "epoch", "of", "belief,", "it", "was", "the", "epoch", "of", "incredulity,", "it", "was", "the", "season", "of", "Light,", "it", "was", "the", "season", "of", "Darkness,", "it", "was", "the", "spring", "of", "hope,", "it", "was", "the", "winter", "of", "despair,", "we", "had", "everything", "before", "us,", "we", "had", "nothing", "before", "us,", "we", "were", "all", "going", "direct", "to", "Heaven,", "we", "were", "all", "going", "direct", "the", "other", "way--in", "short,", "the", "period", "was", "so", "far", "like", "the", "present", "period,", "that", "some", "of", "its", "noisiest", "authorities", "insisted", "on", "its", "being", "received,", "for", "good", "or", "for", "evil,", "in", "the", "superlative", "degree", "of", "comparison", "only."}
