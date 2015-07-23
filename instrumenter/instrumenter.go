//Dinv - instrumenter is a static analysis tool and code modification
//tool for go code. The Instrumenter injects logging code into existing go
//source files. The injected code logs variable values at the point of
//injection, along with the line number and vector clock corresponding
//to the time of the logging.

//modified : july 9 2015 - Stewart Grant

package instrumenter

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"

	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	"bitbucket.org/bestchai/dinv/programslicer"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
)

var (
	logger *log.Logger
)

//Settings houses all of the instrumenters configiguable options
type Settings struct {
	dataflow bool
	debug    bool
}

//defineSettings allows the
//TODO put settings into dinv.go
func defineSettings() *Settings {
	return &Settings{
		dataflow: false,
		debug:    true,
	}
}

//Instrument oversees the instrumentation of an entire package
//for each file provided
//TODO take a package name rather then a single file
func Instrument(dir, packageName string, inlogger *log.Logger) {
	logger = inlogger
	logger.Printf("INSTRUMENTING FILES %s for package %s", dir, packageName)
	settings := defineSettings()
	program := initializeInstrumenter(dir, packageName)
	writeInjectionFile(program.packageName)
	//TODO rename i to source
	for sourceFile := range program.source {
		genCode := generateCode(program, sourceFile, settings)
		instrumented := injectCode(program, sourceFile, genCode)
		writeInstrumentedFile(instrumented, "mod_", program.source[sourceFile].filename)
	}
}

//initializeInstrumenter builds cfg's based on the source location,
//it must be run before other functions, it also returns the source of
//the program
func initializeInstrumenter(dir, packageName string) *ProgramWrapper {
	// Create the AST by parsing src.
	program := getWrappers(dir, packageName)

	for i := range program.source {
		buf := new(bytes.Buffer)
		printer.Fprint(buf, program.fset, program.source[i].comments)

		program.source[i].text = buf.String()
	}
	return program
}

//generateCode constructs code for dump statements for the source code
//located at program.source[sourceIndex].
func generateCode(program *ProgramWrapper, sourceIndex int, settings *Settings) []string {
	var generated_code []string
	var collectedVariables []string
	dumpNodes := GetDumpNodes(program.source[sourceIndex].comments) //test
	for _, dump := range dumpNodes {
		dumpPos := dump.Pos()
		//file relitive dump position (dump abs - file abs = dump rel)
		fileRelitiveDumpPosition := int(dumpPos - program.source[sourceIndex].comments.Pos() + 1)
		lineNumber := program.fset.Position(dumpPos).Line
		if settings.dataflow {
			collectedVariables = getAccessedAffectedVars(dump, program)
		} else {
			collectedVariables = GetAccessibleVarsInScope(fileRelitiveDumpPosition, program.source[sourceIndex].comments, program.fset)
		}
		dumpcode := GenerateDumpCode(collectedVariables, lineNumber, program.source[sourceIndex].filename, program.packageName)

		logger.Println(dumpcode)
		generated_code = append(generated_code, dumpcode)
	}
	return generated_code
}

//injectCode replaces dump statements in the source code of
//program.source[sourceIndex] with lines of code defined in
//injectionCode
func injectCode(program *ProgramWrapper, sourceIndex int, injectionCode []string) string {
	count := 0
	rp := regexp.MustCompile("\\/\\/@dump")
	instrumented := rp.ReplaceAllStringFunc(program.source[sourceIndex].text, func(s string) string {
		replacement := injectionCode[count]
		count++
		return replacement
	})
	return instrumented
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
	vars := sliceComputedVariables(program, recvNodes, programslicer.ComputeForwardSlice)
	vars = append(vars, sliceComputedVariables(program, sendNodes, programslicer.ComputeBackwardSlice)...)
	var varNames []string
	for _, variable := range vars {
		varNames = append(varNames, variable.Name())
	}
	return varNames
}

//TODO pass the program wrapper to the program slicer, along with
//indexes to the what is being calculated
func sliceComputedVariables(program *ProgramWrapper, nodes []*ast.Node, computer func(start ast.Stmt, cfg *cfg.CFG, info *loader.PackageInfo, fset *token.FileSet) []ast.Stmt) []*types.Var {
	var affectedVars []*types.Var
	for _, node := range nodes {
		recvStmt := (*node).(ast.Stmt)
		dcl := findFunction(recvStmt, program.source[0].source.Decls)
		logger.Println("receive function") //BUG These dual print statements seemt to be totally corrupting the output
		logger.Println(dcl)
		firstFunc := program.source[0].source.Decls[dcl].(*ast.FuncDecl)
		program.source[0].cfgs[0].cfg = cfg.FromFunc(firstFunc)
		vars := programslicer.GetAffectedVariables(recvStmt, program.source[0].cfgs[0].cfg, program.prog.Created[0], program.fset, computer)
		affectedVars = append(affectedVars, vars...)
	}
	return affectedVars
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
//TODO rename start to dump line of code
func GetAccessibleVarsInScope(start int, file *ast.File, fset *token.FileSet) []string {
	logger.Println("GETTING VARS!!!") //TODO refactor and put into logger
	var results []string
	//TODO refactor global collection into own function
	global_objs := file.Scope.Objects
	for identifier, _ := range global_objs {
		if global_objs[identifier].Kind == ast.Var || global_objs[identifier].Kind == ast.Con { //|| global_objs[identifier].Kind == ast.Typ { //can be used for diving into structs
			logger.Printf("Global Found :%s\n", fmt.Sprintf("%v", identifier))
			results = append(results, fmt.Sprintf("%v", identifier))
		}
	}

	filePos := fset.File(file.Package)
	if filePos == nil {
		logger.Println("unable to locate dump statement")
	}
	logger.Printf("packagename : %s\n searching Pos start %d\n", file.Name.String(), start)
	//TODO rename path and write comments
	//TODO make the dump location relative to
	path, _ := astutil.PathEnclosingInterval(file, filePos.Pos(start), filePos.Pos(start+2)) // why +2
	for _, node := range path {
		logger.Printf("node at position :%d\n", int(node.Pos()))
	}

	for _, astnode := range path {
		//logger.Println("%v", astutil.NodeDescription(astnode))
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
						logger.Printf("Local Found :%s\n", fmt.Sprintf("%v", identifier))
						results = append(results, fmt.Sprintf("%v", identifier.Name))
					}
				}
			}
		}
	}

	return results
}

//GetDumpNodes traverses a file and returns all ast.Node's
//representing comments of the form //@dump
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
func GenerateDumpCode(vars []string, lineNumber int, path, packagename string) string {
	if len(vars) == 0 {
		return ""
	}
	_, nameWithExt := filepath.Split(path)
	ext := filepath.Ext(path)
	filename := strings.Replace(nameWithExt, ext, "", 1)
	var buffer bytes.Buffer
	// write vars' values
	buffer.WriteString(fmt.Sprintf("\nInstrumenterInit()\n"))
	buffer.WriteString(fmt.Sprintf("%s_%s_%d_vars := []interface{}{", packagename, filename, lineNumber))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("%s,", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("%s}\n", vars[len(vars)-1]))
	// write vars' names
	buffer.WriteString(fmt.Sprintf("%s_%s_%d_varname := []string{", packagename, filename, lineNumber))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("\"%s\",", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("\"%s\"}\n", vars[len(vars)-1]))
	buffer.WriteString(fmt.Sprintf("p%s_%s_%d := CreatePoint(%s_%s_%d_vars, %s_%s_%d_varname, \"%s\",\"%s\",%d)\n",
		packagename, filename, lineNumber,
		packagename, filename, lineNumber,
		packagename, filename, lineNumber,
		packagename, filename, lineNumber))
	buffer.WriteString(fmt.Sprintf("Encoder.Encode(p%s_%s_%d)\n", packagename, filename, lineNumber))
	//write out human readable log
	buffer.WriteString(fmt.Sprintf("ReadableLog.WriteString(p%s_%s_%d.String())", packagename, filename, lineNumber))
	return buffer.String()
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
	logger.Printf("Writing file %s\n", modFilename)
	file.WriteString(source)
	file.Close()
}

//TODO move to its own file
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
//injection code is used to dynamicly write an injection file,
//containing methods called by dump statements

//header_code contains all the needed imports for the injection code,
//and is designed to have the package name written at runtime
var header_code string = `

package %s

import (
	"encoding/gob"
	"os"
	"reflect"
	"strconv"
	"time"
	"fmt"
	"bitbucket.org/bestchai/dinv/govec/vclock"	//attempt to remove dependency
)

var Encoder *gob.Encoder //global
var ReadableLog *os.File
var packageName = "%s"
`

//body code contains utility functions called by the code injected at
//dump statements
//TODO add comments to the inject code
//TODO build array of acceptable types for encoding
//TODO make the logger an argument to CreatePoint
var body_code string = `

func InstrumenterInit() {
	if Encoder == nil {
		stamp := time.Now()
		encodedLogname := fmt.Sprintf("%s-%dEncoded.txt",packageName,stamp.Nanosecond())
		encodedLog, _ := os.Create(encodedLogname)
		Encoder = gob.NewEncoder(encodedLog)
		
		humanReadableLogname := fmt.Sprintf("%s-%dReadable.txt",packageName,stamp.Nanosecond())
		ReadableLog, _ = os.Create(humanReadableLogname)
	}
}

func CreatePoint(vars []interface{}, varNames []string, packagename string, file string, line int) Point {
	numVars := len(varNames)
	dumps := make([]NameValuePair, 0)
	for i := 0; i < numVars; i++ {
		if vars[i] != nil && ((reflect.TypeOf(vars[i]).Kind() == reflect.String) || (reflect.TypeOf(vars[i]).Kind() == reflect.Int)) {
			var dump NameValuePair
			dump.VarName = varNames[i]
			dump.Value = vars[i]
			dump.Type = reflect.TypeOf(vars[i]).String()
			dumps = append(dumps, dump)
		}
	}
	
	point := Point{dumps, strconv.Itoa(line), file, packagename,Logger.GetCurrentVC()}
	return point
}

type Point struct {
	Dump        []NameValuePair
	Line	string
	File	string
	Package string
	VectorClock []byte
}

type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}

func (nvp NameValuePair) String() string {
	return fmt.Sprintf("(%s,%s,%s)", nvp.VarName, nvp.Value, nvp.Type)
}

func (p Point) String() string {
	clock, _ := vclock.FromBytes(p.VectorClock)
	return fmt.Sprintf("%s-%s-%s\n%s %s\nVClock : %s\n\n", p.Package,p.File, p.Line, p.Dump, clock.ReturnVCString())
}`

//TODO move structs to seperate file remove duplication in log merger
