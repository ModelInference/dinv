package logmerger

import "testing"

var logfiles []string = []string{"../TestPrograms/t2/client.go.txt", "../TestPrograms/t2/server.go.txt"}

func TestLattice(t *testing.T) {

	logs := buildLogs(logfiles)
	cases := []struct {
		in   [][]Point
		want int
	}{
		{logs, 0},
	}
	for _, c := range cases {
		got := ConsistantCuts(logs)
		if got != c.want {
			t.Error("MineConsistnat cuts failed")
		}
	}
}
