package ricartagrawala_test

import (
	"testing"
	"flag"
	"time"
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

func TestHalfHostsOneCritical(t *testing.T){
	plan := ricartagrawala.Plan{idInput,0}
	if (idInput%2) == 0 {
		plan.Criticals = 1
	}
	report := ricartagrawala.Host(idInput,hostsInput,plan)
	if !report.ReportMatchesPlan(plan) {
		t.Error(report.ErrorMessage)
	}
}
