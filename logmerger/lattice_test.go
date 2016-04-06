package logmerger

import (
	"testing"

	"github.com/arcaneiceman/GoVector/govec/vclock"
)

func Test2DLattice(t *testing.T) {
	cases := []struct {
		in, want [][]vclock.VClock
	}{
		{
			[][]vclock.VClock{
				[]vclock.VClock{
					*ConstructVclock([]string{"h1"}, []int{1}),
					*ConstructVclock([]string{"h1", "h2"}, []int{3, 2}),
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 2}),
					*ConstructVclock([]string{"h1", "h2"}, []int{6, 4}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h2"}, []int{1}),
					*ConstructVclock([]string{"h2"}, []int{2}),
					*ConstructVclock([]string{"h2"}, []int{4}),
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 6}),
				},
			},
			[][]vclock.VClock{
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{1, 1}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{2, 1}),
					*ConstructVclock([]string{"h1", "h2"}, []int{1, 2}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{2, 2}),
					*ConstructVclock([]string{"h1", "h2"}, []int{1, 3}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{3, 2}),
					*ConstructVclock([]string{"h1", "h2"}, []int{2, 3}),
					*ConstructVclock([]string{"h1", "h2"}, []int{1, 4}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 2}),
					*ConstructVclock([]string{"h1", "h2"}, []int{3, 3}),
					*ConstructVclock([]string{"h1", "h2"}, []int{2, 4}),
					*ConstructVclock([]string{"h1", "h2"}, []int{1, 5}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{5, 2}),
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 3}),
					*ConstructVclock([]string{"h1", "h2"}, []int{3, 4}),
					*ConstructVclock([]string{"h1", "h2"}, []int{2, 5}),
					*ConstructVclock([]string{"h1", "h2"}, []int{1, 6}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{5, 3}),
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 4}),
					*ConstructVclock([]string{"h1", "h2"}, []int{3, 5}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{5, 4}),
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 5}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{6, 4}),
					*ConstructVclock([]string{"h1", "h2"}, []int{5, 5}),
					*ConstructVclock([]string{"h1", "h2"}, []int{4, 6}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{6, 5}),
					*ConstructVclock([]string{"h1", "h2"}, []int{5, 6}),
				},
				[]vclock.VClock{
					*ConstructVclock([]string{"h1", "h2"}, []int{6, 6}),
				},
			},
		},
	}
	for _, c := range cases {
		l := BuildLattice(c.in)
		for i := range l {
			for j := range l[i] {
				if !l[i][j].Compare(&c.want[i][j], vclock.Equal) {
					t.Errorf("Incorrect lattice point \n wanted: %s\n found:%s\n", c.want[i][j].ReturnVCString(), l[i][j].ReturnVCString())
				}
			}
		}
	}
}
