package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/dinvRT"
	"bitbucket.org/bestchai/dinv/capture"
)

const (
	LARGEST_TERM = 100
	RUNS         = 5
)

func main() {

	localAddr, err := net.ResolveUDPAddr("udp4", ":18585")
	printErrAndExit(err)
	remoteAddr, err := net.ResolveUDPAddr("udp4", ":9090")
	printErrAndExit(err)
	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	printErrAndExit(err)

	for t := 1; t <= RUNS; t++ {
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

func reqSum(conn *net.UDPConn, n, m int) (sum int64, err error) {
	msg := make([]byte, 256)
	binary.PutVarint(msg[:8], int64(n))
	binary.PutVarint(msg[8:], int64(m))
	fmt.Println(msg)

	// after instrumentation
	_, err = capture.Write(conn.Write, msg[:])
	_, err = conn.Write(msg)
	if err != nil {
		return
	}

	dinvRT.Dump("main_client_52_", "main_client_52_LARGEST_TERM,main_client_52_RUNS,main_client_52_conn,main_client_52_n,main_client_52_m,main_client_52_msg", LARGEST_TERM, RUNS, conn, n, m, msg)

	buf := make([]byte, 256)
	// after instrumentation
	_, err = capture.Read(conn.Read, buf[:])
	// _, err = conn.Read(buf)
	if err != nil {
		return
	}

	sum, _ = binary.Varint(buf[0:])

	//fmt.Println(sum, buf)

	dinvRT.Dump("main_client_66_", "main_client_66_LARGEST_TERM,main_client_66_RUNS,main_client_66_conn,main_client_66_n,main_client_66_m,main_client_66_msg", LARGEST_TERM, RUNS, conn, n, m, msg)

	return
}

func printErrAndExit(err error) {
	if err != nil {
		fmt.Printf("[CLIENT] %s\n" + err.Error())
		os.Exit(1)
	}
}
