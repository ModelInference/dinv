// ProgramSlicer
package programslicer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
	"bitbucket.org/bestchai/dinv/programslicer/dataflow"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"
)

// Create CFG
// For given graph compute dominator and post-dominators
// Create Control Dependence Graph
// Create Data Dependence Graph
// Create Program Dependence Graph

var (
	debug = true
)

func ComputeForwardSlice(start ast.Stmt, cf *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet) []ast.Stmt {
	fmt.Println("Computing Forward Slice")

	var slice []ast.Stmt
	dataflow.CreateDataDepGraph(cf, info)
	//added
	cf.InitializeBlocks()
	cfg.BuildDomTree(cf)
	invC := cf
	//\/added
	//invC := cf.BuildPostDomTree()		//MODIFIED from post dom
	invC.FindControlDeps()

	if debug {
		var buf bytes.Buffer
		invC.PrintControlDepDot(&buf, fset, func(s ast.Stmt) string {
			if _, ok := s.(*ast.AssignStmt); ok {
				return "!"
			} else {
				return ""
			}
		})

		dot := buf.String()
		fmt.Println(dot)
	}

	visited := make(map[ast.Stmt]bool)
	queue := make([]ast.Stmt, 0)
	queue = append(queue, start)
	visited[start] = true
	slice = append(slice, start)
	for len(queue) != 0 {
		uStmt := queue[0]
		queue = queue[1:]
		u := invC.Blocks[uStmt]

		if debug {
			fmt.Println("visiting ")
			fmt.Println(fset.Position(u.Stmt.Pos()).Line)
			fmt.Println("control deps :")
		}
		for _, v := range u.ControlDepee {

			fmt.Println(fset.Position(v.Stmt.Pos()).Line)
			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
		fmt.Println("data deps :")
		for _, v := range u.DataDepee {
			fmt.Println(fset.Position(v.Stmt.Pos()).Line)
			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
	}
	fmt.Printf("Statements Found %d\n", len(slice))
	return slice
}

func ComputeBackwardSlice(start ast.Stmt, cfg *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet) []ast.Stmt {
	fmt.Println("Computing Backwards Slice")
	var slice []ast.Stmt
	dataflow.CreateDataDepGraph(cfg, info)
	invC := cfg.BuildPostDomTree()
	invC.FindControlDeps()

	if debug {
		var buf bytes.Buffer
		invC.PrintControlDepDot(&buf, fset, func(s ast.Stmt) string {
			if _, ok := s.(*ast.AssignStmt); ok {
				return "!"
			} else {
				return ""
			}
		})

		dot := buf.String()
		fmt.Println(dot)
	}

	visited := make(map[ast.Stmt]bool)
	queue := make([]ast.Stmt, 0)
	queue = append(queue, start)
	visited[start] = true
	slice = append(slice, start)
	for len(queue) != 0 {
		uStmt := queue[0]
		queue = queue[1:]
		u := invC.Blocks[uStmt]

		if debug {
			fmt.Println("visiting ")
			fmt.Println(fset.Position(u.Stmt.Pos()).Line)
			fmt.Println("control deps :")
		}

		for _, v := range u.ControlDep {

			fmt.Println(fset.Position(v.Stmt.Pos()).Line)
			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
		fmt.Println("data deps :")
		for _, v := range u.DataDep {
			fmt.Println(fset.Position(v.Stmt.Pos()).Line)
			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
	}
	fmt.Printf("Statements Found %d\n", len(slice))
	return slice
}

func GetAffectedVariables(start ast.Stmt, cfg *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet, computer func(start ast.Stmt, cfg *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet) []ast.Stmt) []*types.Var {
	stmts := computer(start, cfg, info, fset)
	defs, _ := dataflow.ReferencedVars(stmts, info)

	var affectedVars []*types.Var
	for def := range defs {
		affectedVars = append(affectedVars, def)
	}

	return affectedVars
}

// func main() {
// 	src := `
//   package main
//   func foo(c int, nums []int) int {
//     //START
//     a := c      //6
//     var b int   //7
//     b += 1      //8
//     c, a = a, c //9
//     b = a       //10
//     for a, c = range nums { //11
//       b += a    //12
//     } // 13
//     a, c = c, a //14
//     c = b       //15
//     b++         //16
//     return a    //17
//     //END
//   }`
// 	src = `
// 	package main

// func main() {
// 	sum := 0
// 	i := 1
// 	for i < 11 {
// 		sum = sum + i
// 		i = i + 1
// 	}
// 	sum++
// 	i++
// }
// `

// 	var config loader.Config
// 	f, err := config.ParseFile("testing", src)
// 	if err != nil {
// 		return // probably don't proceed
// 	}
// 	config.CreateFromFiles("testing", f)
// 	prog, err := config.Load()
// 	if err != nil {
// 		return
// 	}

// 	funcOne := f.Decls[0].(*ast.FuncDecl)
// 	c := cfg.FromFunc(funcOne)
// 	//in, out := dataflow.ReachingDefs(c, prog.Created[0])

// 	//ast.Inspect(f, func(n ast.Node) bool {
// 	//	switch stmt := n.(type) {
// 	//	case ast.Stmt:
// 	//		ins, _ := in[stmt], out[stmt]
// 	//		fmt.Println(len(ins))
// 	//		// do as you please
// 	//	}
// 	//	return true
// 	//})
// 	//var buf bytes.Buffer
// 	//dataflow.CreateDataDepGraph(c, prog.Created[0])
// 	//c.PrintDataDepDot(&buf, prog.Fset, func(s ast.Stmt) string {
// 	//	if _, ok := s.(*ast.AssignStmt); ok {
// 	//		return "!"
// 	//	} else {
// 	//		return ""
// 	//	}
// 	//})
// 	//dot := buf.String()
// 	//fmt.Println(dot)

// 	ast.Print(prog.Fset, funcOne.Body.List)
// 	fmt.Println("computing slice ...")
// 	slice := GetForwardAffectedVariables(funcOne.Body.List[1], c, prog.Created[0], prog.Fset)
// 	fmt.Println("slice")
// 	fmt.Println(len(slice))
// 	for _, s := range slice {
// 		//fmt.Println(prog.Fset.Position(s.Pos()).Line)
// 		fmt.Println(s.Name())
// 	}

// }
