// ProgramSlicer
package cdgTest

import (
	"./cfg"
	"fmt"
	//"github.com/godoctor/godoctor/analysis/dataflow"
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
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
	if c == 1 {
		c = 2
		c = c * c
		if c == 4 {
			c = 43
			c = c + c
		}
	} else {
		c = 10
	}
	c = c + c
    //END
  }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	funcOne := f.Decls[0].(*ast.FuncDecl)
	c := cfg.FromFunc(funcOne)
	_ = c.GetBlocks() // for 100% coverage ;)
	invC := c.BuildPostDomTree()
	var buf bytes.Buffer
	invC.PrintDot(&buf, fset, func(s ast.Stmt) string {
		if _, ok := s.(*ast.AssignStmt); ok {
			return "!"
		} else {
			return ""
		}
	})
	dot := buf.String()
	fmt.Println(invC.BlockSlice)
	cfg.PrintDomTreeDot(&buf, invC, fset)
	dot = buf.String()
	fmt.Println(dot)
	invC.FindControlDeps()
	invC.PrintControlDepDot(&buf, fset, func(s ast.Stmt) string {
		if _, ok := s.(*ast.AssignStmt); ok {
			return "!"
		} else {
			return ""
		}
	})

	dot = buf.String()
	fmt.Println(dot)

	//createMyCFG(c)

}
