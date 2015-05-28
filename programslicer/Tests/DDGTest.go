// ProgramSlicer
package main

import (
	"./cfg"
	"./dataflow"
	"bytes"
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/loader"
)

// Create CFG
// For given graph compute dominator and post-dominators
// Create Control Dependence Graph
// Create Data Dependence Graph
// Create Program Dependence Graph

func main() {
	src := `
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
    c = b       //15
    b++         //16
    return a    //17
    //END
  }`

	var config loader.Config
	f, err := config.ParseFile("testing", src)
	if err != nil {
		return // probably don't proceed
	}
	config.CreateFromFiles("testing", f)
	prog, err := config.Load()
	if err != nil {
		return
	}

	funcOne := f.Decls[0].(*ast.FuncDecl)
	c := cfg.FromFunc(funcOne)
	//in, out := dataflow.ReachingDefs(c, prog.Created[0])

	//ast.Inspect(f, func(n ast.Node) bool {
	//	switch stmt := n.(type) {
	//	case ast.Stmt:
	//		ins, _ := in[stmt], out[stmt]
	//		fmt.Println(len(ins))
	//		// do as you please
	//	}
	//	return true
	//})
	var buf bytes.Buffer
	dataflow.CreateDataDepGraph(c, prog.Created[0])
	c.PrintDataDepDot(&buf, prog.Fset, func(s ast.Stmt) string {
		if _, ok := s.(*ast.AssignStmt); ok {
			return "!"
		} else {
			return ""
		}
	})
	dot := buf.String()
	fmt.Println(dot)

}
