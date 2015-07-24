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
	"strconv"

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
func defaultSettings() *Settings {
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
	settings := initializeInstrumenter()
	program := getPackageWrapper(dir, packageName)
	program = generateSourceText(program)
	writeInjectionFile(program.packageName)
	for sourceFile := range program.source {
		genCode := generateCode(program, sourceFile, settings)
		instrumented := injectCode(program, sourceFile, genCode)
		writeInstrumentedFile(instrumented, "mod_", program.source[sourceFile].filename)
	}
}

//initalizeInstrumenter generates a logger if none exists, and returns
//the default settings
func initializeInstrumenter() *Settings {
	if logger == nil {
		logger = log.New(os.Stdout, "instrumenter:", log.Lshortfile)
	}
	return defaultSettings()
}

func generateSourceText(program *ProgramWrapper) *ProgramWrapper {
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
	dumpNodes := GetDumpNodes(program.source[sourceIndex].comments)
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

func getGlobalVariables(file *ast.File, fset *token.FileSet) []string {
	var results []string
	global_objs := file.Scope.Objects
	for identifier, _ := range global_objs {
		//get variables of type constant and Var
		if global_objs[identifier].Kind == ast.Var || global_objs[identifier].Kind == ast.Con { //|| global_objs[identifier].Kind == ast.Typ { //can be used for diving into structs
			logger.Printf("Global Found :%s\n", fmt.Sprintf("%v", identifier))
			results = append(results, fmt.Sprintf("%v", identifier))
		}
	}
	return results
}

//GetAccessibleVarsInScope returns the variables names of all
//varialbes in scope at the point start.
//TODO rename start to dump line of code
func GetAccessibleVarsInScope(dumpPosition int, file *ast.File, fset *token.FileSet) []string {
	logger.Println("Collecting Scope Variables")
	globals := getGlobalVariables(file, fset)
	locals := getLocalVariables(dumpPosition, file, fset)
	return append(globals, locals...)
}

func getLocalVariables(dumpPosition int, file *ast.File, fset *token.FileSet) []string {
	var results []string
	filePos := fset.File(file.Package)
	logger.Printf("packagename : %s\n searching Pos dumpPosition %d\n", file.Name.String(), dumpPosition)
	//TODO rename path and write comments
	//the +2 is probably to grab a send or receive after the dump??
	path, _ := astutil.PathEnclosingInterval(file, filePos.Pos(dumpPosition), filePos.Pos(dumpPosition+2)) // why +2
	for _, astnode := range path {
		logger.Println("%v", astutil.NodeDescription(astnode))
		switch t := astnode.(type) {
		case *ast.BlockStmt:
			stmts := t.List
			logger.Printf("Block found at position :%d of size %d\n", int(t.Pos()), len(stmts))
			for _, stmtnode := range stmts {
				//logger.Printf("Statement type:%s", stmtnode.)
				switch t := stmtnode.(type) {
				case *ast.DeclStmt:
					idents := t.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names
					for _, identifier := range idents {
						//collect node if in scope at the dump statement
						if int(identifier.Pos()) < dumpPosition {
							logger.Printf("Local Found :%s\n", fmt.Sprintf("%v", identifier))
							results = append(results, fmt.Sprintf("%v", identifier.Name))
						}
					}
				//collect variables from definition assignments
				case *ast.AssignStmt:
					if t.Tok == token.DEFINE {
						for _, exp := range t.Lhs {
							ast.Inspect(exp, func(n ast.Node) bool {
								switch resolvedNode := n.(type) {
								case *ast.Ident:
									if int(resolvedNode.Pos()) < dumpPosition {
										logger.Printf("Local Found :%s\n", resolvedNode.Name)
										results = append(results, resolvedNode.Name)
									}
								}
								return true
							})
						}
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
	id := packagename + "_" + filename + "_" + strconv.Itoa(lineNumber)
	buffer.WriteString(fmt.Sprintf("\nInstrumenterInit()\n"))
	buffer.WriteString(fmt.Sprintf("%s_vars := []interface{}{", id))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("%s,", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("%s}\n", vars[len(vars)-1]))
	// write vars' names
	buffer.WriteString(fmt.Sprintf("%s_varname := []string{", id))
	for i := 0; i < len(vars)-1; i++ {
		buffer.WriteString(fmt.Sprintf("\"%s\",", vars[i]))
	}
	buffer.WriteString(fmt.Sprintf("\"%s\"}\n", vars[len(vars)-1]))
	buffer.WriteString(fmt.Sprintf("p%s := CreatePoint(%s_vars, %s_varname, \"%s\")\n", id, id, id, id))
	buffer.WriteString(fmt.Sprintf("Encoder.Encode(p%s)\n", id))
	//write out human readable log
	buffer.WriteString(fmt.Sprintf("ReadableLog.WriteString(p%s.String())", id))
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
