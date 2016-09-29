// ProgramSlicer
package programslicer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
	"bitbucket.org/bestchai/dinv/programslicer/dataflow"

	"go/types"
	"golang.org/x/tools/go/loader"

	"gopkg.in/eapache/queue.v1"

	"go/printer"
)

// Create CFG
// For given graph compute dominator and post-dominators
// Create Control Dependence Graph
// Create Data Dependence Graph
// Create Program Dependence Graph

var (
	debug = true
)

type FuncNode struct {
	Name    string
	Slice   []ast.Stmt
	Calls   map[*ast.FuncDecl]*FuncNode
	NVars   []*types.Var //networking variables
	Root    ast.Stmt
	Tainted bool
}

func NewFuncNode(fd *ast.FuncDecl, root ast.Stmt, tainted bool) *FuncNode {
	fn := new(FuncNode)
	fn.Name = fd.Name.String()
	fn.Slice = make([]ast.Stmt, 0)
	fn.Calls = make(map[*ast.FuncDecl]*FuncNode, 0)
	fn.NVars = make([]*types.Var, 0)
	fn.Root = root
	fn.Tainted = tainted
	return fn
}

type Task struct {
	Root  ast.Stmt
	FDecl *ast.FuncDecl
}

func NewTask(root ast.Stmt, fd *ast.FuncDecl) *Task {
	return &Task{root, fd}
}

func getExitPoints(fd *ast.FuncDecl, p *ProgramWrapper) []ast.Stmt {
	points := make([]ast.Stmt, 0)
	ast.Inspect(fd, func(n ast.Node) bool {
		switch s := n.(type) {
		case ast.Stmt:
			switch s.(type) {
			case *ast.ReturnStmt:
				points = append(points, s)
			}
		}
		return true
	})
	return points

}
func getEntryPoints(fd *ast.FuncDecl, p *ProgramWrapper, tArgs []int) []ast.Stmt {
	points := make([]ast.Stmt, 0)
	tParams := make([]string, 0)
	//collect the function parameters
	for i := range tArgs {
		name := fd.Type.Params.List[0].Names[tArgs[i]].Name
		fmt.Println(name)
		tParams = append(tParams, name)
	}

	for _, param := range tParams {
		mark := true
		ast.Inspect(fd, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				found := false
				for i := range s.Rhs {
					ast.Inspect(s.Rhs[i], func(r ast.Node) bool {
						switch m := r.(type) {
						case *ast.Ident:
							if m.Name == param {
								found = true
							}
						}
						return true
					})
				}
				if found && mark {
					points = append(points, s)
				}
				found = false
				for i := range s.Lhs {
					ast.Inspect(s.Lhs[i], func(r ast.Node) bool {
						switch m := r.(type) {
						case *ast.Ident:
							if m.Name == param {
								found = true
							}
						}
						return true
					})
				}
				if found {
					mark = false
				}

			}
			return true
		})
	}
	return points

}

//searchFunction is a wrapper for find function,
// it searches through an entire.Program looing for a function, it
// returns the packagenumber,Sourcenumber,function number and a
// pointer to the function if it is found
// otherwise it returns -1 for everything
func searchFunction(root ast.Stmt, p *ProgramWrapper) (int, int, int, *ast.FuncDecl) {
	for pnum, pack := range p.Packages {
		for snum, Source := range pack.Sources {
			fnum, f := findFunction(root, Source.Source.Decls)
			if fnum != -1 {
				return pnum, snum, fnum, f
			}
		}
	}
	return -1, -1, -1, nil
}

//findFunction searches through a set of declaractions decls, for the
//statement stmt, the number of the function, which contains the stmt
//is returned
func findFunction(stmt ast.Stmt, decls []ast.Decl) (int, *ast.FuncDecl) {
	fcount := -1
	for dcl := 0; dcl < len(decls); dcl++ {
		_, ok := decls[dcl].(*ast.FuncDecl)
		if ok {
			fcount++
		}
		if stmt.Pos() > decls[dcl].Pos() && stmt.Pos() < decls[dcl].End() {
			return fcount, decls[dcl].(*ast.FuncDecl)
		}
	}
	return -1, nil
}

//findCallStmnets takes an object as an argument, this object is
//supposed to be a function. Returned is the list of statements in
//which that function was called
func findCallStmnts(call *ast.Object, p *ProgramWrapper) []ast.Stmt {
	callNodes := make([]ast.Stmt, 0)
	for _, pack := range p.Packages {
		for _, Source := range pack.Sources {
			if call != nil {

				if debug {
					fmt.Printf("searching for call %s in source %s\n", call.Name, Source.Filename)
				}
			}
			ast.Inspect(Source.Source, func(n ast.Node) bool {
				switch w := n.(type) {
				case *ast.AssignStmt:
					for _, expr := range w.Rhs {
						switch x := expr.(type) {
						case *ast.CallExpr:
							switch y := x.Fun.(type) {
							case *ast.Ident:
								callNodes = checkCallNode(y, w, call, callNodes)
							}
						}
					}
					break
				case *ast.ExprStmt:
					switch x := w.X.(type) {
					case *ast.CallExpr:
						switch y := x.Fun.(type) {
						case *ast.Ident:
							callNodes = checkCallNode(y, w, call, callNodes)
						}
					}
					break
				}
				return true
			})
		}
	}
	if callNodes == nil {
		fmt.Println("Function %s is never called", call.Name)
	}
	return callNodes
}

func checkCallNode(y *ast.Ident, w ast.Stmt, call *ast.Object, callNodes []ast.Stmt) []ast.Stmt {
	//TODO checking by name does not prove a correlation at the
	//interpackage level
	if y != nil && call != nil {
		if y.Name == call.Name {
			callNodes = append(callNodes, w)
			fmt.Printf("function :%s\t, called\n", y.Name)
		}
	}
	return callNodes
}

func findCalledFunctions(stmts []ast.Stmt, p *ProgramWrapper) ([]*ast.CallExpr, []*ast.FuncDecl, []ast.Stmt) {
	calledFunctionDecs := make([]*ast.FuncDecl, 0)
	callingStatements := make([]ast.Stmt, 0)
	callingExpr := make([]*ast.CallExpr, 0)
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			for _, expr := range s.Rhs {
				switch x := expr.(type) {
				case *ast.CallExpr:
					switch y := x.Fun.(type) {
					case *ast.Ident:
						if y.Obj != nil && y.Obj.Decl != nil {
							switch dec := y.Obj.Decl.(type) {
							case *ast.FuncDecl:
								calledFunctionDecs = append(calledFunctionDecs, dec)
								callingStatements = append(callingStatements, stmt)
								callingExpr = append(callingExpr, x)
							}
						}
					}
				}
			}
		}
	}
	return callingExpr, calledFunctionDecs, callingStatements
}

func getArgs(stmt ast.Stmt) []string {
	names := make([]string, 0)
	ast.Inspect(stmt, func(n ast.Node) bool {
		switch c := n.(type) {
		case *ast.CallExpr:
			for _, a := range c.Args {
				switch v := a.(type) {
				case *ast.Ident:
					names = append(names, v.Name)
					break
				default:
					names = append(names, "NOT_A_NAME")
					break
				}
			}
		}
		return true
	})
	return names
}

func GetTaintedPointsBackwards(slice []ast.Stmt, info *loader.PackageInfo, calledAt ast.Stmt, f *ast.FuncDecl, p *ProgramWrapper) []ast.Stmt {
	if f.Type.Results.NumFields() == 0 {
		return nil
	}
	points := getExitPoints(f, p)
	return points
}
func GetTaintedPointsForward(slice []ast.Stmt, info *loader.PackageInfo, calledAt ast.Stmt, f *ast.FuncDecl, p *ProgramWrapper) []ast.Stmt {
	//get the tainted variables
	defs, _ := dataflow.ReferencedVars(slice, info)
	varNames := make([]string, 0)
	for def := range defs {
		varNames = append(varNames, def.Name())
	}
	//match tainted variabes with function arguments
	args := getArgs(calledAt)
	tArgs := taintedArgs(args, varNames)
	if f.Type.Params.NumFields() == 0 {
		return nil
	}
	points := getEntryPoints(f, p, tArgs)
	return points
}

func ComputeSliceIP(root ast.Stmt, p *ProgramWrapper,
	slicer func(start ast.Stmt, cf *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet, sliceSoFar []ast.Stmt) []ast.Stmt,
	pointCollecter func(slice []ast.Stmt, info *loader.PackageInfo, calledAt ast.Stmt, f *ast.FuncDecl, p *ProgramWrapper) []ast.Stmt) map[*ast.FuncDecl]*FuncNode {

	taskQueue := queue.New()
	info := p.Prog.Created[0] //limited to a single packge TODO map to the package being used
	fs := make(map[*ast.FuncDecl]*FuncNode)
	_, _, _, funcDec := searchFunction(root, p)
	fs[funcDec] = NewFuncNode(funcDec, root, true)
	taskQueue.Add(NewTask(root, funcDec))

	for taskQueue.Length() > 0 {
		task := taskQueue.Peek().(*Task)
		taskQueue.Remove()
		pnum, snum, fnum, fd := searchFunction(task.Root, p)
		if fd == nil {
			fmt.Printf("Error:\t function discriptor not found")
			continue
		} else {
			if debug {
				fmt.Printf("Function %s found with id %d\n", fd.Name, fd)
			}
		}
		stmts := slicer(task.Root, p.Packages[pnum].Sources[snum].Cfgs[fnum].Cfg, info, p.Fset, nil)
		if debug {
			fmt.Printf("Slice Found with %d statements\n", len(stmts))
			fmt.Printf("Current Task:\t Perform Slice in Function %s, From Line %d\n", fd.Name.Name, p.Fset.Position(task.Root.Pos()).Line)
		}
		appended := false

		for _, stm := range stmts {
			//if the function slice does not contain the given
			//statements append them
			if !contains(fs[fd].Slice, stm) {
				fs[fd].Slice = append(fs[fd].Slice, stm)
				appended = true
			}
		}
		if !appended {
			continue
		}

		call, decs, onStmts := findCalledFunctions(fs[fd].Slice, p)

		for i, d := range decs {

			if debug {
				fmt.Printf("Function:\t %s calls %s\n", fs[fd].Name, d.Name)
			}
			///Experimential IPC
			assignment := GenCallJoin(call[i], d)
			if assignment != nil {
				//fmt.Println("Appending a joining assignment for the two functions")
				fs[fd].Slice = append(fs[fd].Slice, assignment)
			}
		}

		//tunnel into functions that this one calls
		for i, f := range decs {

			_, ok := fs[f]
			//if not add the function
			if !ok {
				//if debfmt.Printf("Function %s Added to the function store with value %d \n",f.Name.Name,f)
				fs[f] = NewFuncNode(f, task.Root, false)
			}
			fs[fd].Calls[f] = fs[f]

			points := pointCollecter(fs[fd].Slice, info, onStmts[i], f, p)

			//fmt.Printf("function %s added to the task queue with %d tainted points\n",f.Name,len(points))
			for _, point := range points {
				taskQueue.Add(NewTask(point, f))
			}
		}

		//add functions tainted by calling this one
		if fs[fd].Tainted {
			calledAt := findCallStmnts(fd.Name.Obj, p)
			for _, stmt := range calledAt {
				_, _, _, f := searchFunction(stmt, p)
				if f != nil {
					//fmt.Println(f.Name.Name)
					_, ok := fs[f]
					//if not add the function
					if !ok {
						fs[f] = NewFuncNode(f, task.Root, false)
					}
					fs[f].Tainted = fs[fd].Tainted
				} else {
					fmt.Println("function is nil")
				}
				taskQueue.Add(NewTask(stmt, f))
			}
		}

	}
	return fs

}

//TEST FUNCTION
func GenCallJoin(call *ast.CallExpr, function *ast.FuncDecl) *ast.AssignStmt {
	assignment := new(ast.AssignStmt)
	assignment.Tok = token.EQL
	if function.Type.Params == nil {

		return nil
	}
	for _, params := range function.Type.Params.List {
		for _, names := range params.Names {
			assignment.Lhs = append(assignment.Lhs, names)
		}
	}
	assignment.Rhs = call.Args

	//DEBUGING
	fmt.Printf("PRINTING NEW ASSIGNMENT\n")
	fset := token.NewFileSet()
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, assignment)
	fmt.Println(buf.String())

	return assignment
}

func contains(list []ast.Stmt, item ast.Stmt) bool {
	for _, e := range list {
		if e.Pos() == item.Pos() {
			return true
		}
	}
	return false
}

//returns the index of tainted arguments
func taintedArgs(args, tainted []string) []int {
	taintedargs := make([]int, 0)
	for i, arg := range args {
		for _, taint := range tainted {
			if arg == taint {
				//fmt.Println(i)
				taintedargs = append(taintedargs, i)
			}
		}
	}
	return taintedargs
}

func ComputeForwardSlice(start ast.Stmt, cf *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet, sliceSoFar []ast.Stmt) []ast.Stmt {
	if debug {
		fmt.Println("Computing Forward Slice")
	}
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
	if len(sliceSoFar) != 0 {
		queue = append(queue, sliceSoFar...)
	}
	//fmt.Printf("AST stmnt queue len %d\n, CFG Blocks len %d\n",len(queue),len(invC.Blocks)) //dprint
	visited[start] = true
	slice = append(slice, start)
	for len(queue) != 0 {
		uStmt := queue[0]
		queue = queue[1:]
		u := invC.Blocks[uStmt]

		if u == nil {
			continue
		}

		if debug {
			fmt.Printf("visiting ")
			fmt.Printf("control depsee size :%d\n", len(u.ControlDepee))
		}
		for _, v := range u.ControlDepee {

			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
		for _, v := range u.DataDepee {
			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
	}
	//fmt.Printf("Statements Found %d\n", len(slice))
	return slice
}

func ComputeBackwardSlice(start ast.Stmt, Cfg *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet, sliceSoFar []ast.Stmt) []ast.Stmt {
	if debug {
		fmt.Println("Computing Backwards Slice")
	}
	var slice []ast.Stmt
	dataflow.CreateDataDepGraph(Cfg, info)
	invC := Cfg.BuildPostDomTree()
	invC.FindControlDeps()

	if debug {
		var buf bytes.Buffer
		invC.PrintControlDepDot(&buf, fset, func(s ast.Stmt) string {
			if _, ok := s.(*ast.AssignStmt); ok { //check for dumps and write imports here
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
	if len(sliceSoFar) != 0 {
		queue = append(queue, sliceSoFar...)
	}
	visited[start] = true
	slice = append(slice, start)
	for len(queue) > 0 {
		uStmt := queue[0]
		queue = queue[1:]
		u := invC.Blocks[uStmt]

		//TODO solve why u == nil probably comment ast
		if u == nil {
			continue
		}

		if debug {
			fmt.Printf("visiting %d\t trying %s\n", uStmt.Pos(), u.String())
			fmt.Println(fset.Position(u.Stmt.Pos()).Line)
			fmt.Printf("control deps size :%d\n", len(u.ControlDep))
		}
		for _, v := range u.ControlDep {

			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
		for _, v := range u.DataDep {
			if !visited[v.Stmt] {
				visited[v.Stmt] = true
				queue = append(queue, v.Stmt)
				slice = append(slice, v.Stmt)
			}
		}
	}
	return slice
}

//a waky waky function of functions
func GetAffectedVariables(root ast.Stmt, p *ProgramWrapper,
	slicer func(start ast.Stmt, cf *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet, sliceSoFar []ast.Stmt) []ast.Stmt,
	pointCollecter func(slice []ast.Stmt, info *loader.PackageInfo, calledAt ast.Stmt, f *ast.FuncDecl, p *ProgramWrapper) []ast.Stmt) map[*ast.FuncDecl]*FuncNode {

	nodes := ComputeSliceIP(root, p, slicer, pointCollecter)
	for _, node := range nodes {
		defs, _ := dataflow.ReferencedVars(node.Slice, p.Prog.Created[0])
		for def := range defs {
			node.NVars = append(node.NVars, def)
		}
	}
	return nodes
}
