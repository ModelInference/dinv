package main

import (
	"bitbucket.org/bestchai/dinv/examples/sum/marshall"
	"fmt"
	"math/rand"
	"net"
	"os"
)

const (
	LARGEST_TERM	= 100
	RUNS		= 500
)

func main() {
	localAddr, err := net.ResolveUDPAddr("udp4", ":18585")
	printErrAndExit(err)
	remoteAddr, err := net.ResolveUDPAddr("udp4", ":9090")
	printErrAndExit(err)
	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	printErrAndExit(err)

	for t := 0; t < RUNS; t++ {
		n, m := rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM
		sum, err := reqSum(conn, n, m)
		if err != nil {
			fmt.Printf("[CLIENT] %s", err.Error())
			continue
		}
		fmt.Printf("[CLIENT] %d/%d: %d + %d = %d\n", t, RUNS, n, m, sum)
	}
	fmt.Println()
	os.Exit(0)
}

func reqSum(conn *net.UDPConn, n, m int) (sum int, err error) {
	msg := marshall.MarshallInts([]int{n, m})

	_, err = conn.Write(msg)
	if err != nil {
		return
	}
	//@dump

	var buf [1 * marshall.IntSize]byte
	_, err = conn.Read(buf[0:])
	if err != nil {
		return
	}

	ret := marshall.UnmarshallInts(buf[0:])
	sum = ret[0]

	//@dump

	return
}

func printErrAndExit(err error) {
	if err != nil {
		fmt.Printf("[CLIENT] %s\n" + err.Error())
		os.Exit(1)
	}
}
