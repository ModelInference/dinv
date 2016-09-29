package ricartagrawala_test

import (
	"bitbucket.org/bestchai/dinv/examples/ricartagrawala"
	"flag"
	"fmt"
	"testing"
)

var (
	idInput    int
	hostsInput int
	timeInput  int
)

func TestMain(m *testing.M) {
	var idarg = flag.Int("id", 0, "hosts id")
	var hostsarg = flag.Int("hosts", 0, "#of hosts")
	var timearg = flag.Int("time", 0, "timeout")
	flag.Parse()
	idInput = *idarg
	hostsInput = *hostsarg
	timeInput = *timearg
	m.Run()
}

func TestAllHostsManyCriticals(t *testing.T) {
	plan := ricartagrawala.Plan{idInput, 10, timeInput}
	report := ricartagrawala.Host(idInput, hostsInput, plan)
	if !report.ReportMatchesPlan(plan) {
		fmt.Println("FAILED")
		t.Error(report.ErrorMessage)
	}
	fmt.Println("PASSED")
}
