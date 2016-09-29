// ProgramSlicer
package programslicer

import (
	"bitbucket.org/bestchai/dinv/programslicer/dataflow"
	"bytes"
	"fmt"
	"go/printer"
	"go/token"
	"go/types"
	"strings"
	"testing"
	//"go/ast"
)

// Create CFG
// For given graph compute dominator and post-dominators
// Create Control Dependence Graph
// Create Data Dependence Graph
// Create Program Dependence Graph

func TestForwardIPC(t *testing.T) {
	program, err := GetWrapperFromString(source1)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	node := program.Packages[0].Sources[0].Cfgs[1].Exp[2] //e++
	//ast.Print(program.Fset,program.Packages[0].Sources[0].Source);
	funcStatements := ComputeSliceIP(node, program, ComputeForwardSlice, GetTaintedPointsForward)
	//after slicing print vars
	var affectedVars []*types.Var
	expected := []string{
		"e", "touchMe", "a", "b", "p", "b", "c", "win", "smile", "f",
	}
	for _, stmts := range funcStatements {
		fset := token.NewFileSet()
		var buf bytes.Buffer
		printer.Fprint(&buf, fset, stmts.Slice)
		fmt.Println(buf.String())
		println()

		defs, _ := dataflow.ReferencedVars(stmts.Slice, program.Prog.Created[0])

		for def := range defs {
			affectedVars = append(affectedVars, def)
		}
	}
	matchCheck(expected, affectedVars, t)
}

func TestBackwardsIPCBasic(t *testing.T) {
	program, err := GetWrapperFromString(source2)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	node := program.Packages[0].Sources[0].Cfgs[0].Exp[1] //a := bar()
	//ast.Print(program.Fset,program.Packages[0].Sources[0].Source);
	funcStatements := ComputeSliceIP(node, program, ComputeBackwardSlice, GetTaintedPointsBackwards)
	//after slicing print vars
	var affectedVars []*types.Var
	expected := []string{
		"a", "b",
	}
	for _, stmts := range funcStatements {
		defs, _ := dataflow.ReferencedVars(stmts.Slice, program.Prog.Created[0])

		for def := range defs {
			affectedVars = append(affectedVars, def)
		}
		//should print a, b
	}
	matchCheck(expected, affectedVars, t)
}

func TestBackwardsIPCDontTouchCoo(t *testing.T) {
	program, err := GetWrapperFromString(source3)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	node := program.Packages[0].Sources[0].Cfgs[0].Exp[1] //a := bar()
	//ast.Print(program.Fset,program.Packages[0].Sources[0].Source);
	funcStatements := ComputeSliceIP(node, program, ComputeBackwardSlice, GetTaintedPointsBackwards)
	//after slicing print vars
	var affectedVars []*types.Var
	expected := []string{
		"a", "b",
	}
	for _, stmts := range funcStatements {
		defs, _ := dataflow.ReferencedVars(stmts.Slice, program.Prog.Created[0])

		for def := range defs {
			affectedVars = append(affectedVars, def)
		}
	}
	matchCheck(expected, affectedVars, t)
}

/*
func TestBackwardsIPCPassByReference(t *testing.T) {
	program, err := GetWrapperFromString(source4);
	if err != nil {
		t.Errorf("Error: %s",err.Error())
	}

	node := program.Packages[0].Sources[0].Cfgs[1].Exp[1] //f := b
	ast.Print(program.Fset,program.Packages[0].Sources[0].Source);
	funcStatements := ComputeSliceIP(node,program,ComputeBackwardSlice,GetTaintedPointsBackwards)
	//after slicing print vars
	for _, stmts := range funcStatements {
		defs, _ := dataflow.ReferencedVars(stmts.Slice, program.Prog.Created[0])

		var affectedVars []*types.Var
		for def := range defs {
			affectedVars = append(affectedVars, def)
		}
		//should print a, b
		fmt.Printf("Vars: %s", affectedVars);
	}
}
*/

func matchCheck(expected []string, found []*types.Var, t *testing.T) {
	for i := range expected {
		present := false
		for j := range found {
			//print(strings.Compare(expected[i],found[j].Name()))
			if strings.Compare(expected[i], found[j].Name()) == 0 {
				present = true
			}
		}
		if !present {
			for j := range found {
				print(found[j].Name() + "\n")
			}
			t.Errorf("variable %s not found when expected\nFound%s\n", expected[i])
		}
	}
	for i := range found {
		included := false
		for j := range expected {
			if found[i].Name() == expected[j] {
				included = true
			}
		}
		if !included {
			t.Errorf("variable %s found, but not expected\n", found[i].Name())
		}
	}
}

const source1 = `
  package main

  func foo(c int, nums []int) int {
    //START
    a := c      //6
    var b int   //7
    b += 1      //8
    c, a = a, c //9
    b = a       //10
    for a, c = range nums { //11
      b += a    //12
    } // 13
    a, c = c, a //14
	b, c = bar(b, c)//15
	b, c = bar(b, c)//15
	//ipcdataflow
	var win int
	var pin int
	win = pin
	win = b
	win++
	var smile int 
	smile = win
	win = smile
	pin++
	smile = pin //pin should not be collected
    return a    //16
    //END
  }

  
  func bar(e, f int) ( int, int ) {
	//START BAR
	f = e
	e++
	e = deep(e)
	return e, f
	}

  func barPlus(p int) int {
	p, p = bar(p, p)
	return p
  }

  func deep(touchMe int) int {
	  	a := touchMe
		touchMe = 1
		b := touchMe
		print(b)
		return a
	}

	func deepTouch(){
		touched := 1
		touched = deep(touched)
	}
  
  `

const source2 = `package main		//1
					//2
	func foo() {	//3
		a := bar()	//4
		print(a)	//5
	}				//6
					//7
	func bar() int {//8
		b:= 2		//9
		return b	//10
	}				//11
  `

//V shapped call graph foo()--->bar()<----coo()
const source3 = `
	package main

	func foo() {
		a := bar() //touch here
		print(a)
	}

	func bar() int {
		b:= 2
		return b
	}

	func coo() {
		c := bar()
		print(c)
	}
`

const source4 = `
	package main

 	func foo() {
		var a int
		a = 5
		c := a
		bar(&c)
	}

	func bar( b *int) {
		*b = *b + 1
		f := *b		//starting statemnt
		print(f)
	 }
	`
