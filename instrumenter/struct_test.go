package instrumenter

import (
	"go/ast"
	"testing"
)

const structSource = `
package main
	var identity t
	var spec s
	func main(argc int){
		truth := false
		identity.alive = truth
		identity.age = 35
		spec.species = "jackalope"
		spec.condition = identity
		//@dump
	}
	type t struct {
		alive bool
		age int
	}
	type s struct {
		condition t
		species string
	}
	`

var wantVars [][]string = [][]string{
	[]string{"identity", "spec", "identity.alive", "identity.age", "spec.condition", "spec.condition.alive", "spec.condition.age", "spec.species", "truth"},
}

var structOptions map[string]string = map[string]string{
	"debug": "",
	"file":  "",
}

func TestStructUnwrap(t *testing.T) {
	initializeInstrumenter(structOptions, nil)
	program, err := getWrapperFromString(structSource)
	if err != nil {
		t.Fatal(err)
	}
	ast.Print(program.fset, program.packages[0].sources[0].comments)
	dumpNodes := GetDumpNodes(program.packages[0].sources[0].comments)
	for i := range dumpNodes {
		fileRelitiveDumpPosition := int(dumpNodes[i].Pos() - program.packages[0].sources[0].comments.Pos() + 1)
		collectedVariables := GetAccessibleVarsInScope(fileRelitiveDumpPosition, program.packages[0].sources[0].comments, program.fset)
		if !matchSet(wantVars[i], collectedVariables) {
			t.Errorf("failed collection {want:%s | collected:%s}\n", wantVars[i], collectedVariables)
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
