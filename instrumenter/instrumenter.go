package instrumenter

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"

	"regexp"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"

	"bitbucket.org/bestchai/dinv/programslicer"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
)

const (
	START = 0
	END   = 100000000
)

var src_location string
var usage string = "go run instrumenter.go toinstrument > instrumented.go"

var fset *token.FileSet
var astFile *ast.File
var c *CFGWrapper
var wrappers []*CFGWrapper

func Instrument(files []string) {
	fmt.Println("INSTRUMENTING FILES")
	for _, file := range files {
		src_location = file
		optimize := false
		source := initializeInstrumenter()
		dumpNodes := GetDumpNodes(astFile)
		var generated_code []string
		if !optimize {
			fmt.Println("GETTING VARS 1")
			for _, dump := range dumpNodes {
				line := c.fset.Position(dump.Pos()).Line
				// log all vars
				//generated_code = append(generated_code, GenerateDumpCode(GetAccessedVarsInScope(dump, c.f, c), line))
				generated_code = append(generated_code, GenerateDumpCode(GetAccessibleVarsInScope(int(dump.Pos()), astFile, c.fset), line))
				fmt.Println(generated_code[0])
			}
		} else {
			for _, dump := range dumpNodes {
				line := c.fset.Position(dump.Pos()).Line
				generated_code = append(generated_code, GenerateDumpCode(getAccessedAffectedVars(dump, astFile, c), line))

			}
		}
		count := 0
		rp := regexp.MustCompile("\\/\\/@dump")
		transformed := rp.ReplaceAllStringFunc(source, func(s string) string {
			replacement := generated_code[count]
			count++
			return replacement
		})

		transformed = transformed + "\n" + extra_code

		rp = regexp.MustCompile("[ ]*func[ ]+main\\(\\)[ ]+{")
		//fmt.Println(transformed)
		insturmented := fmt.Sprintf("%s", rp.ReplaceAllString(transformed, "func main() {\n InstrumenterInit()\n"))
		//fmt.Print(insturmented)
		writeInstrumentedFile(insturmented, file)
		//fmt.Println(detectSendReceive(astFile))
	}
}

func getAccessedAffectedVars(dump *ast.Comment, file *ast.File, cf *CFGWrapper) []string {

	var affectedInScope []string
	inScope := GetAccessibleVarsInScope(int(dump.Pos()), file, cf.fset)
	affected := getAffectedVars(cf)

	for _, inScopeVar := range inScope {
		for _, affectedVar := range affected {
			if inScopeVar == affectedVar {
				affectedInScope = append(affectedInScope, inScopeVar)
				break
			}
		}
	}

	return affectedInScope

}

func findFunction(stmt ast.Stmt, cf *CFGWrapper) int {
	for dcl := 0; dcl < len(cf.f.Decls)-1; dcl++ {
		if stmt.Pos() > cf.f.Decls[dcl].Pos() && stmt.Pos() < cf.f.Decls[dcl+1].Pos() {
			return dcl
		}
	}
	return -1
}

func getAffectedVars(cf *CFGWrapper) []string {
	recvNodes := detectFunctionCalls(cf.f, "conn", []string{"Read", "ReadFrom"})
	sendNodes := detectFunctionCalls(cf.f, "conn", []string{"Write", "WriteTo"})

	for i := 0; i < len(recvNodes)+len(sendNodes); i++ {
		fmt.Printf("%d,", i)
	}

	//fmt.Println(recvNodes)
	//fmt.Println(sendNodes)
	var affectedVars []*types.Var

	for _, node := range recvNodes {
		recvStmt := (*node).(ast.Stmt)
		dcl := findFunction(recvStmt, cf)
		fmt.Println("receive function") //BUG These dual print statements seemt to be totally corrupting the output
		fmt.Println(dcl)
		firstFunc := cf.f.Decls[dcl].(*ast.FuncDecl)
		cf.cfg = cfg.FromFunc(firstFunc)
		vars := programslicer.GetAffectedVariables(recvStmt, cf.cfg, cf.prog.Created[0], cf.prog.Fset, programslicer.ComputeForwardSlice)
		affectedVars = append(affectedVars, vars...)
	}

	for _, node := range sendNodes {
		sendStmt := (*node).(ast.Stmt)

		dcl := findFunction(sendStmt, cf)
		fmt.Println("send function")
		fmt.Println(dcl)
		firstFunc := cf.f.Decls[dcl].(*ast.FuncDecl)
		cf.cfg = cfg.FromFunc(firstFunc)
		vars := programslicer.GetAffectedVariables(sendStmt, cf.cfg, cf.prog.Created[0], cf.prog.Fset, programslicer.ComputeBackwardSlice)

		affectedVars = append(affectedVars, vars...)
	}
	var affectedVarName []string
	for _, variable := range affectedVars {
		affectedVarName = append(affectedVarName, variable.Name())
	}
	return affectedVarName
}

//initializeInstrumenter builds cfg's based on the source location,
//it must be run before other functions, it also returns the source of
//the program
//TODO is a really bad function and requires way too many globals,
//lets turf it at some point
func initializeInstrumenter() string {
	extra_code = template_code
	extra_code = fmt.Sprintf(extra_code, src_location)
	// Create the AST by parsing src.
	fset = token.NewFileSet() // positions are relative to fset
	astFile, _ = parser.ParseFile(fset, src_location, nil, parser.ParseComments)
	wrappers := getWrappers(nil, src_location)
	c = wrappers[0] //TODO this is an artifact from only one function being analyized delete this eventually

	addImports(astFile)

	var buf bytes.Buffer
	printer.Fprint(&buf, fset, astFile)

	s := buf.String()
	//print(s)
	return s
}

//replacement for match send and receive
//detectFunctionCalls searches an ast.File for instances of a varible
//(varname) making a calls to a list of functions. Nodes in the ast
//where such calls are made are returned
//ex. conn.Write, and conn.WriteTo are searchable
func detectFunctionCalls(f *ast.File, varName string, funcNames []string) []*ast.Node {
	var results []*ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		switch z := n.(type) {
		case *ast.ExprStmt:
			switch x := z.X.(type) {
			case *ast.CallExpr:
				if matchCallExpression(x, varName, funcNames) {
					results = append(results, &n)
				}
			}
		case *ast.AssignStmt:
			switch x := z.Rhs[0].(type) {
			case *ast.CallExpr:
				if matchCallExpression(x, varName, funcNames) {
					results = append(results, &n)
				}
			}
			return true
		}
		return true
	})
	return results
}

//matchCallExpression determines if a particular call expression
//involvs (varName) calling any of the listed functions
func matchCallExpression(n *ast.CallExpr, varName string, funcNames []string) bool {
	switch y := n.Fun.(type) {
	case *ast.SelectorExpr:
		left, _ := y.X.(*ast.Ident)
		if left.Name == varName {
			for _, name := range funcNames {
				if y.Sel.Name == name {
					return true
				}
			}
		}
	}
	return false
}

/*
func GetAccessedVarsInScope(dumpNode *ast.Comment, f *ast.File, cf *CFGWrapper) []string {
	var results []string
	path, _ := astutil.PathEnclosingInterval(f, dumpNode.Pos(), dumpNode.End())

	var stmts []ast.Stmt
	var unwantedVars []string

	//print("inspecting")
	for _, astnode := range path {
		//print("node")
		funcDecl, ok := astnode.(*ast.FuncDecl)
		if ok { // skip import decl if exists

			ast.Inspect(funcDecl, func(n ast.Node) bool {
				switch x := n.(type) {
				case ast.Stmt:
					switch x.(type) {
					case *ast.BlockStmt:
						return true

					//begin assignment checking
					case *ast.AssignStmt:
						astmnt := x.(*ast.AssignStmt)

						if astmnt.Tok.String() == ":=" {
							var localStmts []ast.Stmt
							ast.Inspect(x, func(r ast.Node) bool {
								print("s")
								switch s := r.(type) {
								case ast.Stmt:
									if (s.(ast.Stmt)).Pos() < dumpNode.Pos() {
										localStmts = append(localStmts, s)
									}
								}
								return true
							})
								defs, _ := dataflow.ReferencedVars(localStmts, cf.prog.Created[0])
								for d, _ := range defs {
									scope := d.Parent
									names := scope.Names()
									for _, name := range names {
										fmt.Print(name)
									}
									fmt.Println()
									unwantedVars = append(unwantedVars, d.Name())
								}
						}
						return true

						//end assignment checking
					}
					if x.Pos() < dumpNode.Pos() {
						stmts = append(stmts, x)
					}
				case *ast.FuncLit:
					// skip statements in anonymous functions
					return false
				}
				return true
			})
		}

	}
	_, uses := dataflow.ReferencedVars(stmts, cf.prog.Created[0])

	//actualUse := make(map[*types.Var]struct{})
	for u, _ := range uses {
		results = append(results, u.Name())
	}

	//test
	print("unwanted\n")
	for _, name := range unwantedVars {
		fmt.Println(name)
	}
	//for _, result := range results {
	//	fmt.Printf("%s\n", result)
	//}

	return results

}
*/

func GetAccessibleVarsInScope(start int, file *ast.File, fset *token.FileSet) []string {
	fmt.Println("GETTING VARS!!!")
	var results []string
	global_objs := file.Scope.Objects
	for identifier, _ := range global_objs {
		if global_objs[identifier].Kind == ast.Var || global_objs[identifier].Kind == ast.Con { //|| global_objs[identifier].Kind == ast.Typ { //can be used for diving into structs
			fmt.Printf("Global Found :%s\n", fmt.Sprintf("%v", identifier))
			fmt.Printf("Checking for struct Type")
			structure, ok := global_objs[identifier].Decl.(*ast.StructType)
			if ok {
				for _, fields := range structure.Fields.List {
					fmt.Println("found Field Name", fields.Names[0].Name)
				}
			}
			results = append(results, fmt.Sprintf("%v", identifier))
		}
	}
	filePos := fset.File(file.Package)
	path, _ := astutil.PathEnclosingInterval(file, filePos.Pos(start), filePos.Pos(start+2))

	for _, astnode := range path {
		//fmt.Println("%v", astutil.NodeDescription(astnode))
		switch t := astnode.(type) {
		case *ast.BlockStmt:

			stmts := t.List
			for _, stmtnode := range stmts {
				switch t := stmtnode.(type) {
				case *ast.DeclStmt:
					idents := t.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names
					for _, identifier := range idents {
						fmt.Printf("Local Found :%s\n", fmt.Sprintf("%v", identifier))
						results = append(results, fmt.Sprintf("%v", identifier.Name))
					}
				}
			}
		}
	}

	return results
}

func GetDumpNodes(file *ast.File) []*ast.Comment {
	var dumpNodes []*ast.Comment
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			if strings.Contains(comment.Text, "@dump") {
				dumpNodes = append(dumpNodes, comment)
			}
		}
	}
	return dumpNodes
}

// returns dump code that should replace that specific line number
func GenerateDumpCode(vars []string, lineNumber int) string {
	if len(vars) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	// write vars' values
	buffer.WriteString(fmt.Sprintf("InstrumenterInit()\n"))
	buffer.WriteString(fmt.Sprintf("vars%d := []interface{}{", lineNumber))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("%s,", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("%s}\n", vars[len(vars)-1]))
	// write vars' names
	buffer.WriteString(fmt.Sprintf("varsName%d := []string{", lineNumber))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("\"%s\",", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("\"%s\"}\n", vars[len(vars)-1]))
	buffer.WriteString(fmt.Sprintf("point%d := createPoint(vars%d, varsName%d, %d)\n", lineNumber, lineNumber, lineNumber, lineNumber))
	buffer.WriteString(fmt.Sprintf("encoder.Encode(point%d)", lineNumber))
	return buffer.String()
}

var extra_code string
var template_code string = `

var encoder *gob.Encoder

func InstrumenterInit() {
	if encoder == nil {
		fileW, _ := os.Create("%s.txt")
		encoder = gob.NewEncoder(fileW)
	}
}

func createPoint(vars []interface{}, varNames []string, lineNumber int) Point {

	length := len(varNames)
	dumps := make([]NameValuePair, 0)
	for i := 0; i < length; i++ {

		if vars[i] != nil && ((reflect.TypeOf(vars[i]).Kind() == reflect.String) || (reflect.TypeOf(vars[i]).Kind() == reflect.Int)) {
			var dump NameValuePair
			dump.VarName = varNames[i]
			dump.Value = vars[i]
			dump.Type = reflect.TypeOf(vars[i]).String()
			dumps = append(dumps, dump)
		}
	}
	
	point := Point{dumps, strconv.Itoa(lineNumber), Logger.GetCurrentVC()}
	return point
}

type Point struct {
	Dump        []NameValuePair
	LineNumber  string
	VectorClock []byte
}

type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}

//func (nvp NameValuePair) String() string {
//	return fmt.Sprintf("(%s,%s,%s)", nvp.VarName, nvp.Value, nvp.Type)
//}

//func (p Point) String() string {
//	return fmt.Sprintf("%s : %s", p.LineNumber, p.Dump)
//}
`

func addImports(file *ast.File) {
	packagesToImport := []string{"\"encoding/gob\"", "\"os\"", "\"reflect\"", "\"strconv\""}
	im := ImportAdder{packagesToImport}
	ast.Walk(im, file)
}

type ImportAdder struct {
	PackagesToImport []string
}

func (im ImportAdder) Visit(node ast.Node) (w ast.Visitor) {
	switch t := node.(type) {
	case *ast.GenDecl:
		if t.Tok == token.IMPORT {
			//remove duplicate imports
			releventImports := nonDuplicateImports(im.PackagesToImport, t.Specs)
			newSpecs := make([]ast.Spec, len(t.Specs)+len(releventImports))
			for i, spec := range t.Specs {
				newSpecs[i] = spec
			}
			for i, spec := range releventImports {
				newPackage := &ast.BasicLit{token.NoPos, token.STRING, spec}
				newSpecs[len(t.Specs)+i] = &ast.ImportSpec{nil, nil, newPackage, nil, token.NoPos}
			}

			t.Specs = newSpecs
			return nil
		}
	}
	return im
}

func nonDuplicateImports(packagesToImport []string, specs []ast.Spec) []string {
	var releventImports []string
	for _, potential := range packagesToImport {
		var duplicate bool = false
		for _, existing := range specs {
			enode := existing.(ast.Node)
			switch e := enode.(type) {
			case *ast.ImportSpec:
				if potential == e.Path.Value { //not sure this compairison works
					duplicate = true
					break
				}
			}
		}
		if !duplicate {
			releventImports = append(releventImports, potential)
		}
	}
	return releventImports
}

type CFGWrapper struct {
	cfg      *cfg.CFG
	prog     *loader.Program
	exp      map[int]ast.Stmt
	stmts    map[ast.Stmt]int
	objs     map[string]*types.Var
	objNames map[*types.Var]string
	fset     *token.FileSet
	f        *ast.File
}

func getWrappers(t *testing.T, filename string) []*CFGWrapper {
	var config loader.Config
	//fmt.Println("\n\n" + filename + "\n\n")
	f, err := config.ParseFile(filename, nil)
	files := make([]*ast.File, 0)
	files = append(files, f)
	dir, _ := filepath.Split(filename)
	fmt.Println(dir, filename)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		sdir, _ := filepath.Split(path)
		//add all other files in the same directory
		if ext != ".go" || path == filename || sdir != dir {
			return nil
		}
		fmt.Println(path)
		g, err := config.ParseFile(path, nil)
		if err != nil {
			return nil
		}
		files = append(files, g)
		return nil
	})
	if err != nil {
		fmt.Println("CannotLoad")
		t.Error(err.Error())
		t.FailNow()
		return nil
	}

	config.CreateFromFiles("testing", files...)
	prog, err := config.Load()
	if err != nil {
		fmt.Println("CannotLoad")
		t.Error(err.Error())
		t.FailNow()
		return nil
	}

	cfgs := make([]*CFGWrapper, 0)
	for i := 0; i < len(f.Decls); i++ {
		functionDec, ok := f.Decls[i].(*ast.FuncDecl)
		if ok {
			print("FuncFound\n")
			wrap := getWrapper(t, f, functionDec, prog)
			cfgs = append(cfgs, wrap)
		}
	}
	return cfgs
}

// uses first function in given string to produce CFG
// w/ some other convenient fields for printing in test
// cases when need be...
func getWrapper(t *testing.T, f *ast.File, functionDec *ast.FuncDecl, prog *loader.Program) *CFGWrapper {
	cfg := cfg.FromFunc(functionDec)
	v := make(map[int]ast.Stmt)
	stmts := make(map[ast.Stmt]int)
	objs := make(map[string]*types.Var)
	objNames := make(map[*types.Var]string)
	i := 1
	//fmt.Println("GETTING WRAPPER")
	ast.Inspect(functionDec, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if obj, ok := prog.Created[0].ObjectOf(x).(*types.Var); ok {
				objs[obj.Name()] = obj
				objNames[obj] = obj.Name()
			}
		case ast.Stmt:
			switch x.(type) {
			case *ast.BlockStmt:
				return true
			}
			v[i] = x
			stmts[x] = i
			i++
		case *ast.FuncLit:
			// skip statements in anonymous functions
			return false
		}
		return true
	})
	v[END] = cfg.Exit
	v[START] = cfg.Entry
	stmts[cfg.Entry] = START
	stmts[cfg.Exit] = END
	//if len(v) != len(cfg.GetBlocks()) {
	//	t.Logf("expected %d vertices, got %d --construction error", len(v), len(cfg.GetBlocks()))
	//}
	//fmt.Printf("-----func start print---------------\n")
	//ast.Print(prog.Fset, f)
	//fmt.Printf("-----func end print-----------------\n")

	return &CFGWrapper{
		cfg:      cfg,
		prog:     prog,
		exp:      v,
		stmts:    stmts,
		objs:     objs,
		objNames: objNames,
		fset:     prog.Fset,
		f:        f,
	}
}

//prints given AST
func (c *CFGWrapper) printAST() {
	ast.Print(c.fset, c.f)
}

func writeInstrumentedFile(source string, filename string) {
	pwd, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	_, name := filepath.Split(filename)
	modFilename := fmt.Sprintf("%s/mod_%s", pwd, name)
	file, _ := os.Create(modFilename)
	fmt.Printf("Writing file %s\n", modFilename)
	file.WriteString(source)
	file.Close()
}
