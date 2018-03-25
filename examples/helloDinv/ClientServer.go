package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"bitbucket.org/bestchai/dinv/dinvRT"

	"flag"
	"log"
	"runtime/pprof"
)

const (
	SERVERPORT = "8080"
	CLIENTPORT = "8081"
	MESSAGES   = 100
)

var done chan int = make(chan int, 2)
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var isServer = flag.Bool("isServer", false, "True if the this process is the server")
var isClient = flag.Bool("isClient", false, "True if the this process is the client")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *isServer && *isClient {
		log.Fatal("Both Client and Server mode activated, specify only 1, exiting")
	} else if !*isServer && !*isClient {
		log.Fatal("Neither Client or Server specified, using only 1, exiting")
	}
	if *isServer {
		server(SERVERPORT)
	}
	if *isClient {
		client(CLIENTPORT, SERVERPORT)
	}
}

func client(listen, send string) {
	// sending UDP packet to specified address and port
	conn := setupConnection(SERVERPORT, CLIENTPORT)

	for i := 0; i < MESSAGES; i++ {
		dinvRT.Track("main_ClientServer_55_", "main_ClientServer_55_isServer,main_ClientServer_55_isClient,main_ClientServer_55_SERVERPORT,main_ClientServer_55_MESSAGES,main_ClientServer_55_done,main_ClientServer_55_CLIENTPORT,main_ClientServer_55_cpuprofile,main_ClientServer_55_listen,main_ClientServer_55_send,main_ClientServer_55_conn", isServer, isClient, SERVERPORT, MESSAGES, done, CLIENTPORT, cpuprofile, listen, send, conn)
		outgoingMessage := i
		outBuf := dinvRT.Pack(outgoingMessage)
		_, errWrite := conn.Write(outBuf)
		printErr(errWrite)
		var inBuf [512]byte
		var incommingMessage int
		n, errRead := conn.Read(inBuf[0:])
		printErr(errRead)
		dinvRT.Unpack(inBuf[0:n], &incommingMessage)
		incommingMessage = n - n + incommingMessage
		//fmt.Printf("GOT BACK : %d\n", incommingMessage)
		time.Sleep(1)

	}
	done <- 1

}

func server(listen string) {
	conn, err := net.ListenPacket("udp", ":"+listen)
	printErr(err)

	var buf = make([]byte, 512)

	var n, nMinOne, nMinTwo int

	for i := 0; i < MESSAGES; i++ {
		dinvRT.Track("main_ClientServer_83_", "main_ClientServer_83_isServer,main_ClientServer_83_isClient,main_ClientServer_83_SERVERPORT,main_ClientServer_83_MESSAGES,main_ClientServer_83_done,main_ClientServer_83_CLIENTPORT,main_ClientServer_83_cpuprofile,main_ClientServer_83_listen,main_ClientServer_83_conn,main_ClientServer_83_err,main_ClientServer_83_buf,main_ClientServer_83_n,main_ClientServer_83_nMinOne,main_ClientServer_83_nMinTwo", isServer, isClient, SERVERPORT, MESSAGES, done, CLIENTPORT, cpuprofile, listen, conn, err, buf, n, nMinOne, nMinTwo)
		_, addr, err := conn.ReadFrom(buf[0:])
		dinvRT.Unpack(buf, &n)
		var incommingMessage int
		//fmt.Printf("Recieved %d\n", incommingMessage)
		printErr(err)

		switch incommingMessage {
		case 0:
			nMinTwo = 0
			n = 0
			break
		case 1:
			nMinOne = 0
			n = 1
			break
		default:
			nMinTwo = nMinOne
			nMinOne = n
			n = nMinOne + nMinTwo
			break
		}
		buf := dinvRT.Pack(n)
		conn.WriteTo(buf, addr)
		time.Sleep(1)

	}
	conn.Close()
	done <- 1

}

func setupConnection(sendingPort, listeningPort string) *net.UDPConn {
	rAddr, errR := net.ResolveUDPAddr("udp4", ":"+sendingPort)
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":"+listeningPort)
	printErr(errL)

	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)
	if (errR == nil) && (errL == nil) && (errDial == nil) {
		return conn
	}
	return nil
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
