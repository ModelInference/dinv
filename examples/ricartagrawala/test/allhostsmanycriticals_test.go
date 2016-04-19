package ricartagrawala_test

import (
	"testing"
	"flag"
	"fmt"
	"bitbucket.org/bestchai/dinv/examples/ricartagrawala"
)

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

func TestAllHostsManyCriticals(t *testing.T){
	plan := ricartagrawala.Plan{idInput,10}
	fmt.Println(plan.Criticals)
	report := ricartagrawala.Host(idInput,hostsInput,plan)
	if !report.ReportMatchesPlan(plan) {
		t.Error(report.ErrorMessage)
	}
}
