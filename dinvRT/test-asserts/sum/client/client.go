package main

import (
	"bitbucket.org/bestchai/dinv/dinvRT"
	"encoding/binary"
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"math/rand"
	"net"
	"os"
    "github.com/acarb95/DistributedAsserts/assert"
    "time"
)

const (
	LARGEST_TERM = 100
	RUNS         = 5//00
)

var n int64
var m int64
var sum int64

var client_assert_addr = ":18589"
var server_assert_addr = ":9099"

// ============================== ASSERT CODE ==============================
func assertValue(values map[string]map[string]interface{}) bool {
	int_a := values[server_assert_addr]["a"].(int64)
	int_b := values[server_assert_addr]["b"].(int64)
	int_n := values[client_assert_addr]["n"].(int64)
	int_m := values[client_assert_addr]["m"].(int64)
	int_sum := values[client_assert_addr]["sum"].(int64)

	if (int_n != int_a) && (int_n != int_b) {
		fmt.Println("ASSERTION FAILURE: client n does not match server a or b.")
		fmt.Printf("\tn: %d, a: %d, b: %d\n", int_n, int_a, int_b)
		return false
	} else if (int_m != int_a) && (int_m != int_b) {
	  	fmt.Println("ASSERTION FAILURE: client n does not match server a or b.")
		fmt.Printf("\tm: %d, a: %d, b: %d\n", int_m, int_a, int_b)
		return false
	} else if int_sum != (int_m + int_n) {
		fmt.Println("ASSERTION FAILURE: sum does not match n + m.")
		fmt.Printf("\tsum: %d, n: %d, m: %d\n", int_sum, int_n, int_m)
		return false
	} else {
		return true
	}
}
// ============================ END ASSERT CODE ============================


func main() {
	// ============================== ASSERT CODE ==============================
	assert.InitDistributedAssert(client_assert_addr, []string{server_assert_addr}, "client");
	assert.AddAssertable("n", &n, nil);
	assert.AddAssertable("m", &m, nil);
	assert.AddAssertable("sum", &sum, nil)
	// ============================ END ASSERT CODE ============================

	time.Sleep(5*time.Second)

	localAddr, err := net.ResolveUDPAddr("udp4", ":18585")
	printErrAndExit(err)
	remoteAddr, err := net.ResolveUDPAddr("udp4", ":9090")
	printErrAndExit(err)
	conn, err := net.DialUDP("udp4", localAddr, remoteAddr)
	printErrAndExit(err)

	for t := 1; t <= RUNS; t++ {
		n, m = int64(rand.Int()%LARGEST_TERM), int64(rand.Int()%LARGEST_TERM)
		sum, err = reqSum(conn, n, m)
		if err != nil {
			fmt.Printf("[CLIENT] %s", err.Error())
			continue
		}

		// ============================== ASSERT CODE ==============================
		// Add requested value for current program, then for every other neighbor
		requestedValues := make(map[string][]string)
		requestedValues[client_assert_addr] = append(requestedValues[client_assert_addr], "n")
		requestedValues[client_assert_addr] = append(requestedValues[client_assert_addr], "m")
		requestedValues[client_assert_addr] = append(requestedValues[client_assert_addr], "sum")

		requestedValues[server_assert_addr] = append(requestedValues[server_assert_addr], "a")
		requestedValues[server_assert_addr] = append(requestedValues[server_assert_addr], "b")

		// Assert on those requested things. 
		assert.Assert(assertValue, requestedValues)
		// ============================ END ASSERT CODE ============================


		time.Sleep(assert.GetAssertDelay()*2)
		// fmt.Printf("[CLIENT] %d/%d: %d + %d = %d\n", t, RUNS, n, m, sum)
	}
	fmt.Println()
	os.Exit(0)
}

func reqSum(conn *net.UDPConn, n, m int64) (sum int64, err error) {
	msg := make([]byte, 32) // capture.Write adds to the buffer so this buffer size needs to be smaller than what is read on the server
	binary.PutVarint(msg[:8], n)
	binary.PutVarint(msg[8:], m)

	// after instrumentation
	_, err = capture.Write(conn.Write, msg)
	// fmt.Println(written)
	// _, err = conn.Write(msg)
	if err != nil {
		return
	}

	buf := make([]byte, 256)
	// after instrumentation
	_, err = capture.Read(conn.Read, buf[:])
	// _, err = conn.Read(buf)
	if err != nil {
		return
	}

	dinvRT.Track("client", "n, m, sum", n, m, sum)

	sum, _ = binary.Varint(buf[0:])

	// fmt.Println(sum, buf)

	return
}

func printErrAndExit(err error) {
	if err != nil {
		fmt.Printf("[CLIENT] %s\n" + err.Error())
		os.Exit(1)
	}
}
