package logmerger

import (
	"testing"

	"github.com/arcaneiceman/GoVector/govec/vclock"
)

type want struct {
	clocks []*vclock.VClock
	id     string
	logs   []string
}

type testFramework struct {
	in string
	w  want
}

var govecRegex string = "(\\S*) ({.*})\n(.*)"

//test1 covers the basic parsing case where no errors have occured in
//the log. the log is written in the govec format and is entirely
//correct

func TestParse(t *testing.T) {
	Init()
	case1index1clock := ConstructVclock([]string{"Server"}, []int{1})
	case1index2clock := ConstructVclock([]string{"Server", "Client"}, []int{2, 2})
	cases := []testFramework{
		{in: `Server {"Server":1}
Initialization Complete
Server {"Server":2, "Client":2}
Received`,
			w: want{
				clocks: []*vclock.VClock{case1index1clock, case1index2clock},
				id:     "Server",
				logs:   []string{"Initialization Complete", "Received"},
			},
		},
	}
	for _, c := range cases {
		goLog, err := LogsFromString(c.in, govecRegex)
		if err != nil {
			t.Errorf("parse error : %s", err)
		}
		if goLog.id != c.w.id {
			t.Errorf("Incorrect Id\t want: %s\t got: %s\n")
		}
		if len(goLog.messages) != len(c.w.logs) {
			t.Errorf("Incorrect log length parsed\t want %d\t got %d\n", len(c.w.logs), len(goLog.messages))
		}
		for log := range c.w.logs {
			if goLog.messages[log] != c.w.logs[log] {
				t.Errorf("Inconsistant logs \n want: %s\n got: %s\n", c.w.logs[log], goLog.messages[log])
			}
		}
		if len(goLog.clocks) != len(c.w.clocks) {
			t.Errorf("Incorrect clock length parsed\t want %d\t got %d\n", len(c.w.clocks), len(goLog.clocks))
		}
		for i := range goLog.clocks {
			if !goLog.clocks[i].Compare(c.w.clocks[i], vclock.Equal) {
				t.Errorf("Clock Parse error \n want:%s\n got %s\n", c.w.clocks[i].ReturnVCString(), goLog.clocks[i].ReturnVCString())
			}
		}

	}
}
