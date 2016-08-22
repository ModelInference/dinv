/*

	//TODO refactor to global states
	The state of a distributed program can be represented as the set
	of all variable values across all hosts a moment in time. The
	moment in time is denoted by the Cut, the set of all variable
	values is within the set of points. and the total ordering within
	the cut represents the communication at that point in time.

	Author: Stewart Grant
	Edited: July 6 2015
*/

package logmerger

import "fmt"

//State of a distributed program at a moment corresponding to a cut
type State struct {
	Cut           Cut
	Points        []Point
	TotalOrdering [][]int
}

//String representation of state, the cut is returned aloing with the
//point and all of the coresponding variable values. The total
//ordering is represened as a string matching the host indexes in the
//cut
func (state State) String() string {
	catString := fmt.Sprintf("@ Cut %s \\Cut \n[", state.Cut.String())
	catString = fmt.Sprintf("%s States", catString)
	for i := range state.Points {
		catString = fmt.Sprintf("%s [%s]", catString, state.Points[i].String())
	}
	catString = fmt.Sprintf("%s] \\State\n", catString)
	for i := range state.TotalOrdering {
		catString = fmt.Sprintf("%s[", catString)
		for j := range state.TotalOrdering[i] {
			catString = fmt.Sprintf("%s %d,", catString, state.TotalOrdering[i][j])
		}
		catString = fmt.Sprintf("%s]", catString)
	}
	catString = fmt.Sprintf("%s]@\n", catString)
	return catString
}
