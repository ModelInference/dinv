package instrumenter

import "testing"

const source = `
package main
	var available = "Im Here"
	func main(){
		//@dump
		integer := 5
		other := integer
		other = 6
		//@dump
		for i :=0;i<integer;i++{
			other = integer + other + i
			//@dump
			mother := other
			integer = -mother
			//@dump
		}
		//@dump
	}
	//@dump
	`

var want [][]string = [][]string{
	[]string{"available"},
	[]string{"available", "integer", "other"},
	[]string{"available", "integer", "other"}, //this and next should be adjusted to include i
	[]string{"available", "integer", "other", "mother"},
	[]string{"available", "integer", "other"},
	[]string{"available"},
}

func TestScopeFor(t *testing.T) {
	initializeInstrumenter()
	program, err := getWrapperFromString(source)
	if err != nil {
		t.Fatal(err)
	}
	dumpNodes := GetDumpNodes(program.packages[0].sources[0].comments)
	//ast.Print(program.fset, program.source[0].comments)
	for i := range dumpNodes {
		fileRelitiveDumpPosition := int(dumpNodes[i].Pos() - program.packages[0].sources[0].comments.Pos() + 1)
		collectedVariables := GetAccessibleVarsInScope(fileRelitiveDumpPosition, program.packages[0].sources[0].comments, program.fset)
		if !matchSet(want[i], collectedVariables) {
			t.Errorf("failed collection {want:%s | collected:%s}\n", want[i], collectedVariables)
		}
	}
}

func matchSet(first, second []string) bool {
	for _, varname := range first {
		if !contains(varname, second) {
			return false
		}
	}
	for _, varname := range second {
		if !contains(varname, first) {
			return false
		}
	}
	return true
}

func contains(single string, many []string) bool {
	for _, one := range many {
		if one == single {
			return true
		}
	}
	return false
}
