package main

import (
	"bitbucket.org/bestchai/dinv/dinvRT"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
)

var (
	counter  int64
	neighbor net.Conn
	logger   *log.Logger
)

func main() {
	port := flag.Int("port", 0, "port which this node listens on")
	neighborPort := flag.Int("neighbor", 0, "port neighbor node listens on")

	flag.Parse()

	if *port == 0 || *neighborPort == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	nodeID := "[N" + strconv.Itoa(*port) + "]"

	logger = log.New(os.Stdout, nodeID+" ", log.Lshortfile)

	listenAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(*port))
	printErrAndExit(err)
	neighborAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(*neighborPort))
	printErrAndExit(err)

	listener, err := net.ListenUDP("udp", listenAddr)
	printErrAndExit(err)
	defer listener.Close()

	neighbor, err = net.DialUDP("udp", nil, neighborAddr)
	printErrAndExit(err)

	logger.Printf("listening on %s, neighbor is %s", listenAddr.String(), neighborAddr.String())

	go func() {
		for {
			buf := make([]byte, 8)
			// _, err := listener.Read(buf)
			_, err := capture.Read(listener.Read, buf)
			if err != nil {
				fmt.Println(err)
				continue
			}
			update, _ := binary.Varint(buf)

			logger.Printf("received %v (%d)\n", buf, update)

			if update > atomic.LoadInt64(&counter) {
				atomic.StoreInt64(&counter, update)
				logger.Printf("received update: counter now at %d", atomic.LoadInt64(&counter))

				dinvRT.Dump(nodeID, "counter", atomic.LoadInt64(&counter))
				if err = forward(); err != nil {
					fmt.Println(err)
					continue
				}
			} else {
				logger.Printf("received update %d, but counter already at %d\n", update,
					atomic.LoadInt64(&counter))
			}
		}
	}()

	// hook for bash script
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	for {
		<-sigusr1
		atomic.AddInt64(&counter, 1)
		logger.Printf("signal received: counter increased to %d", atomic.LoadInt64(&counter))
		// when Dump is before forward, invariant N8003-counter - N8002-counter == 0 shows up
		if err := forward(); err != nil {
			logger.Println(err)
		}
		// when Dump is after forward, invariant N8003-counter == N8002-counter shows up
		dinvRT.Dump(nodeID, "counter", atomic.LoadInt64(&counter))
	}
}

func forward() (err error) {
	msg := make([]byte, 8)
	binary.PutVarint(msg, atomic.LoadInt64(&counter))
	_, err = capture.Write(neighbor.Write, msg)
	return
}

func printErrAndExit(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
