package instrumenter

import "testing"

func TestLoad(t *testing.T) {
	cases := []struct {
		in struct {
			t        *testing.T
			filename string
		}
		want error
	}{
		{{t, "insturmenter.go"}, nil},
	}
	for _, c := range cases {
		w := getWrappers(c.in.t, c.in.filename)
	}
}

//t.Error("string")
