package chainkv_test

import (
	"bitbucket.org/bestchai/dinv/examples/chainkv"
	"flag"
	"fmt"
	"testing"
)

const BASEPORT = 8080

var (
	idInput    int
	hostsInput int
	endInput   int
)

func TestMain(m *testing.M) {
	var idarg = flag.Int("id", 0, "hosts id")
	var hostsarg = flag.Int("hosts", 0, "#of hosts")
	var endarg = flag.Int("end", 0, "-------")
	flag.Parse()
	idInput = *idarg
	hostsInput = *hostsarg
	endInput = *endarg
	m.Run()
}

func TestChain(t *testing.T) {
	if idInput == 0 {
		//start the client
		chainkv.Client(fmt.Sprintf("%d", BASEPORT), fmt.Sprintf("%d", BASEPORT+1), fmt.Sprintf("%d", BASEPORT+hostsInput))
		// change endInput to 0 for normal execution and 1 for skipped
		// last node
	} else if idInput < (hostsInput - 1) {
		//start a middle node
		chainkv.Node(fmt.Sprintf("%d", BASEPORT+idInput), fmt.Sprintf("%d", BASEPORT+idInput+1), fmt.Sprintf("%d", BASEPORT+hostsInput))
	} else {
		//start the last node
		chainkv.Node(fmt.Sprintf("%d", BASEPORT+idInput), fmt.Sprintf("%d", BASEPORT), fmt.Sprintf("%d", BASEPORT+hostsInput))
	}
}
