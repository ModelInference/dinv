package dinvRT

import "testing"

var (
	integer int
	words   string
	boolean bool
)

func BenchmarkDumps(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Dump("id", "int,string,bool", integer, words, boolean)
	}
}
