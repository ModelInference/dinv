package chainkv_test

import (
	"testing"
	"flag"
	"fmt"
	"bitbucket.org/bestchai/dinv/examples/chainkv"
)

const BASEPORT = 8080

var (
	idInput int
	hostsInput int
)

func TestMain(m *testing.M) {
	var idarg = flag.Int("id",0, "hosts id")
	var hostsarg = flag.Int("hosts",0, "#of hosts")
	flag.Parse()
	idInput = *idarg
	hostsInput = *hostsarg
	m.Run()
}

func TestChain(t *testing.T){
	if idInput == 0 {
		chainkv.Client(fmt.Sprintf("%d",BASEPORT),fmt.Sprintf("%d",BASEPORT + 1), fmt.Sprintf("%d",BASEPORT + hostsInput))
	} else if idInput < hostsInput {
		chainkv.Node(fmt.Sprintf("%d",BASEPORT + idInput),fmt.Sprintf("%d",BASEPORT + idInput + 1), fmt.Sprintf("%d",BASEPORT + hostsInput))
	} else {
		chainkv.Node(fmt.Sprintf("%d",BASEPORT + idInput),fmt.Sprintf("%d",BASEPORT), fmt.Sprintf("%d",BASEPORT + hostsInput))
	}
}
