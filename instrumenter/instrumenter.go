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

//Instrument oversees the instrumentation of an entire package
//for each file provided
//TODO take a package name rather then a single file
func Instrument(files []string) {
	fmt.Println("INSTRUMENTING FILES")
	for _, file := range files {
		optimize := false
		program := initializeInstrumenter(file)
		writeInjectionFile(program.packageName)
		for i := range program.source {
			dumpNodes := GetDumpNodes(program.source[i].comments)
			var generated_code []string
			if !optimize {
				fmt.Println("GETTING VARS 1")
				for _, dump := range dumpNodes {
					line := program.fset.Position(dump.Pos()).Line
					// log all vars
					//generated_code = append(generated_code, GenerateDumpCode(GetAccessedVarsInScope(dump, c.f, c), line))
					generated_code = append(generated_code, GenerateDumpCode(GetAccessibleVarsInScope(int(dump.Pos()), program.source[i].comments, program.fset), line, program.source[i].filename))
					fmt.Println(generated_code[0])
				}
			} else {
				for _, dump := range dumpNodes {
					line := program.fset.Position(dump.Pos()).Line
					generated_code = append(generated_code, GenerateDumpCode(getAccessedAffectedVars(dump, program), line, program.source[i].filename))
				}
			}
			count := 0
			rp := regexp.MustCompile("\\/\\/@dump")
			insturmented := rp.ReplaceAllStringFunc(program.source[i].text, func(s string) string {
				replacement := generated_code[count]
				count++
				return replacement
			})
			writeInstrumentedFile(insturmented, "mod_", program.source[i].filename)
		}
	}
}

//getAccessedAffectedVars returns the names of all variables affected
//by a send, or a receive within the scope of the dump statement
func getAccessedAffectedVars(dump *ast.Comment, program *ProgramWrapper) []string {

	var affectedInScope []string
	inScope := GetAccessibleVarsInScope(int(dump.Pos()), program.source[0].comments, program.fset)
	affected := getAffectedVars(program)

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

//findFunction searches through a set of declaractions decls, for the
//statement stmt, the index of that statement is returned if it is
//found, if not -1 is returned.
func findFunction(stmt ast.Stmt, decls []ast.Decl) int {
	for dcl := 0; dcl < len(decls)-1; dcl++ {
		if stmt.Pos() > decls[dcl].Pos() && stmt.Pos() < decls[dcl+1].Pos() {
			return dcl
		}
	}
	return -1
}

//getAfffectedVars searches through an entire program specified by
//program, and returns the names of all variables modified by
//interprocess communication.

//TODO getAffectedVars does not work at the moment and should be
//restructured. The variables returned should be thoses affected by
//IPC around a particular dump statement, not the entire program
func getAffectedVars(program *ProgramWrapper) []string {
	recvNodes := detectFunctionCalls(program.source[0].source, "conn", []string{"Read", "ReadFrom"})
	sendNodes := detectFunctionCalls(program.source[0].source, "conn", []string{"Write", "WriteTo"})

	for i := 0; i < len(recvNodes)+len(sendNodes); i++ {
		fmt.Printf("%d,", i)
	}

	//fmt.Println(recvNodes)
	//fmt.Println(sendNodes)
	var affectedVars []*types.Var
	for _, node := range recvNodes {
		recvStmt := (*node).(ast.Stmt)
		dcl := findFunction(recvStmt, program.source[0].source.Decls)
		fmt.Println("receive function") //BUG These dual print statements seemt to be totally corrupting the output
		fmt.Println(dcl)
		firstFunc := program.source[0].source.Decls[dcl].(*ast.FuncDecl)
		program.source[0].cfgs[0].cfg = cfg.FromFunc(firstFunc)
		vars := programslicer.GetAffectedVariables(recvStmt, program.source[0].cfgs[0].cfg, program.prog.Created[0], program.fset, programslicer.ComputeForwardSlice)
		affectedVars = append(affectedVars, vars...)
	}

	for _, node := range sendNodes {
		sendStmt := (*node).(ast.Stmt)

		dcl := findFunction(sendStmt, program.source[0].source.Decls)
		fmt.Println("send function")
		fmt.Println(dcl)
		firstFunc := program.source[0].source.Decls[dcl].(*ast.FuncDecl)
		program.source[0].cfgs[0].cfg = cfg.FromFunc(firstFunc)
		vars := programslicer.GetAffectedVariables(sendStmt, program.source[0].cfgs[0].cfg, program.prog.Created[0], program.fset, programslicer.ComputeForwardSlice)
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
func initializeInstrumenter(src_location string) *ProgramWrapper {
	// Create the AST by parsing src.
	program := getWrappers(nil, src_location)

	for i := range program.source {
		var buf bytes.Buffer
		printer.Fprint(&buf, program.fset, program.source[i].comments)
		program.source[i].text = buf.String()
	}
	//print(s)
	return program
}

//(replacement for match send and receive)
//detectFunctionCalls searches an ast.File for instances of a varible
//(varname) making a calls to a list of functions. Nodes in the ast
//where such calls are made are returned
//ex. conn.Write, and conn.WriteTo are searchable as sending functions
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

//GetAccessibleVarsInScope returns the variables names of all
//varialbes in scope at the point start.
func GetAccessibleVarsInScope(start int, file *ast.File, fset *token.FileSet) []string {
	fmt.Println("GETTING VARS!!!")
	var results []string
	global_objs := file.Scope.Objects
	for identifier, _ := range global_objs {
		if global_objs[identifier].Kind == ast.Var || global_objs[identifier].Kind == ast.Con { //|| global_objs[identifier].Kind == ast.Typ { //can be used for diving into structs
			fmt.Printf("Global Found :%s\n", fmt.Sprintf("%v", identifier))
			results = append(results, fmt.Sprintf("%v", identifier))
		}
	}
	filePos := fset.File(file.Package)
	path, _ := astutil.PathEnclosingInterval(file, filePos.Pos(start), filePos.Pos(start+2)) // why +2

	for _, astnode := range path {
		//fmt.Println("%v", astutil.NodeDescription(astnode))
		switch t := astnode.(type) {
		case *ast.BlockStmt:
			//BUG variables which have yet to be declared are
			//appearing and causing compile time errors
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

//GetDumpNodes traverses a file and returns all comments matching the
//form @dump
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

//GenerateDumpCode constructs code to be injected at dump points, the
//code includes a call to initialize the insturmenter, the packaging
//of all variables and their values, and the encoding of a
//corresponding vector clock
//TODO Removde Dump dependency on global variable "Logger"
func GenerateDumpCode(vars []string, lineNumber int, path string) string {
	if len(vars) == 0 {
		return ""
	}
	_, nameWithExt := filepath.Split(path)
	ext := filepath.Ext(path)
	filename := strings.Replace(nameWithExt, ext, "", 1)
	var buffer bytes.Buffer
	// write vars' values
	buffer.WriteString(fmt.Sprintf("\nInstrumenterInit()\n"))
	buffer.WriteString(fmt.Sprintf("%svars%d := []interface{}{", filename, lineNumber))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("%s,", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("%s}\n", vars[len(vars)-1]))
	// write vars' names
	buffer.WriteString(fmt.Sprintf("%svarsName%d := []string{", filename, lineNumber))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("\"%s\",", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("\"%s\"}\n", vars[len(vars)-1]))
	buffer.WriteString(fmt.Sprintf("%spoint%d := CreatePoint(%svars%d, %svarsName%d, %d, \"%s\")\n", filename, lineNumber, filename, lineNumber, filename, lineNumber, lineNumber, filename))
	buffer.WriteString(fmt.Sprintf("Encoder.Encode(%spoint%d)", filename, lineNumber))
	return buffer.String()
}

type ProgramWrapper struct {
	prog        *loader.Program
	fset        *token.FileSet
	packageName string
	source      []*SourceWrapper
}

type SourceWrapper struct {
	comments *ast.File
	source   *ast.File
	filename string
	text     string
	cfgs     []*CFGWrapper
}

type CFGWrapper struct {
	cfg      *cfg.CFG
	exp      map[int]ast.Stmt
	stmts    map[ast.Stmt]int
	objs     map[string]*types.Var
	objNames map[*types.Var]string
}

func getWrappers(t *testing.T, filename string) *ProgramWrapper {
	fmt.Println("Getting Wrappers")
	var config loader.Config
	//fmt.Println("\n\n" + filename + "\n\n")
	commentFile, _ := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ParseComments)
	commentFiles := make([]*ast.File, 0)
	commentFiles = append(commentFiles, commentFile)

	sourceFile, err := config.ParseFile(filename, nil)
	sourceFiles := make([]*ast.File, 0)
	sourceFiles = append(sourceFiles, sourceFile)

	filenames := make([]string, 0)
	filenames = append(filenames, filename)

	pName := commentFile.Name.String()

	dir, _ := filepath.Split(filename)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		fmt.Println(path)
		ext := filepath.Ext(path)
		sdir, _ := filepath.Split(path)
		//add all other files in the same directory
		if ext != ".go" || path == filename || sdir != dir {
			return nil
		}
		fmt.Println(path)
		source, err := config.ParseFile(path, nil)
		if err != nil {
			return nil
		}
		sourceFiles = append(sourceFiles, source)

		comments, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}
		commentFiles = append(commentFiles, comments)
		filenames = append(filenames, path)
		return nil
	})
	if err != nil {
		fmt.Println("CannotLoad")
		t.Error(err.Error())
		t.FailNow()
		return nil
	}
	fmt.Println("Loading Files")
	config.CreateFromFiles("testing", sourceFiles...)
	prog, err := config.Load()
	if err != nil {
		fmt.Println("CannotLoad")
		t.Error(err.Error())
		t.FailNow()
		return nil
	}
	fmt.Println("Files Loaded")

	sources := make([]*SourceWrapper, 0)
	for i, file := range sourceFiles {
		fmt.Printf("building source for %s\n", filenames[i])
		cfgs := make([]*CFGWrapper, 0)
		for j := 0; j < len(file.Decls); j++ {
			fmt.Printf("building CFG[%d]\n", j)
			functionDec, ok := file.Decls[j].(*ast.FuncDecl)
			if ok {
				print("FuncFound\n")
				wrap := getWrapper(t, functionDec, prog)
				cfgs = append(cfgs, wrap)
			}
		}
		fmt.Println("Source Built")
		sources = append(sources, &SourceWrapper{
			comments: commentFiles[i],
			source:   sourceFiles[i],
			filename: filenames[i],
			cfgs:     cfgs})
	}
	fmt.Println("Wrappers Built")
	return &ProgramWrapper{
		prog:        prog,
		fset:        prog.Fset,
		packageName: pName,
		source:      sources,
	}

}

//getWrapper creates a wrapper for a control flow graph
func getWrapper(t *testing.T, functionDec *ast.FuncDecl, prog *loader.Program) *CFGWrapper {
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

	return &CFGWrapper{
		cfg:      cfg,
		exp:      v,
		stmts:    stmts,
		objs:     objs,
		objNames: objNames,
	}
}

//prints given AST
func (p *ProgramWrapper) printAST() {
	for _, source := range p.source {
		ast.Print(p.fset, source.source)
	}
}

//writeInstrumentedFile writes a file with the contents source, with
//the filename "prefixfilename"
func writeInstrumentedFile(source string, prefix string, filename string) {
	pwd, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	_, name := filepath.Split(filename)
	modFilename := fmt.Sprintf("%s/%s%s", pwd, prefix, name)
	file, _ := os.Create(modFilename)
	fmt.Printf("Writing file %s\n", modFilename)
	file.WriteString(source)
	file.Close()
}

//writeInjectionFile builds a library file that generated code calls.
//The injected file must belong to the same package as the
//insturmented files, specified by packageName the resulting file will
//be created in PWD "mod_inject.go"
func writeInjectionFile(packageName string) {
	header := header_code
	header = fmt.Sprintf(header, packageName, packageName)
	fileString := header + "\n" + body_code
	writeInstrumentedFile(fileString, "mod_", "inject.go")
}

/* Injection Code */
var header_code string = `

package %s

import (
	"encoding/gob"
	"os"
	"reflect"
	"strconv"
	"time"
	"fmt"
)

var Encoder *gob.Encoder
var packageName = "%s"
`

var body_code string = `

func InstrumenterInit() {
	if Encoder == nil {
		stamp := time.Now()
		filename := fmt.Sprintf("%s-%d.txt",packageName,stamp.Nanosecond())
		fileW, _ := os.Create(filename)
		Encoder = gob.NewEncoder(fileW)
	}
}

func CreatePoint(vars []interface{}, varNames []string, lineNumber int, filename string) Point {

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
	
	point := Point{dumps, strconv.Itoa(lineNumber), filename, Logger.GetCurrentVC()}
	return point
}

type Point struct {
	Dump        []NameValuePair
	LineNumber  string
	FileName	string
	VectorClock []byte
}

type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}`
