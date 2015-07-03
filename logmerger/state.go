package logmerger

import "fmt"

type State struct {
	Cut           Cut
	Points        []Point
	TotalOrdering [][]int
}

func (state State) String() string {
	catString := fmt.Sprintf("%s\n[", state.Cut.String())
	for i := range state.Points {
		catString = fmt.Sprintf("%s [%s]", catString, state.Points[i].String)
	}
	catString = fmt.Sprintf("%s]\n", catString)
	for i := range state.TotalOrdering {
		catString = fmt.Sprintf("%s[", catString)
		for j := range state.TotalOrdering[i] {
			catString = fmt.Sprintf("%s %d,", catString, state.TotalOrdering[i][j])
		}
		catString = fmt.Sprintf("%s]", catString)
	}
	catString = fmt.Sprintf("%s]\n", catString)
	return catString
}
