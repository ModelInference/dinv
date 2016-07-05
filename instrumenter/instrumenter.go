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

	"strings"

	"bitbucket.org/bestchai/dinv/programslicer"
	"golang.org/x/tools/go/ast/astutil"
	"github.com/arcaneiceman/GoVector/capture"

)

var (
	logger           *log.Logger
	clean 			 = false
	dataflow         = false
	debug            = false
	dumpsLocalEvents = false
	instDirectory    = ""
	instFile         = ""
)

//Instrument oversees the instrumentation of an entire package
//for each file provided
func Instrument(options map[string]string, inlogger *log.Logger) {
	initializeInstrumenter(options, inlogger)
	program, err := getProgramWrapper()
	if err != nil {
		logger.Fatalf("Error: %s", err.Error())
	}
	if clean {
		cleanSources(program)
		return
	}
	ast.Print(program.Fset, program.Packages[0].Sources[0].Source)
	for pnum, pack := range program.Packages {
		for snum := range pack.Sources {
			InsturmentSource(program, pnum, snum)
			writeInstrumentedFile(program, pnum, snum)
		}
	}
}

//initalizeInstrumenter generates a logger if none exists, and returns
//the default settings
func initializeInstrumenter(options map[string]string, inlogger *log.Logger) {
	if inlogger == nil {
		logger = log.New(os.Stdout, "instrumenter:", log.Lshortfile)
	} else {
		logger = inlogger
	}
	for setting := range options {
		switch setting {
		case "clean":
			clean = true
		case "debug":
			debug = true
		case "dataflow":
			dataflow = true
		case "directory":
			instDirectory = options[setting]
			logger.Printf("Insturmenting Directory :%s", instDirectory)
		case "file":
			instFile = options[setting]
			logger.Printf("Insturmenting File :%s", instFile)
		case "local":
			dumpsLocalEvents = true
		default:
			continue
		}
	}
}


func getProgramWrapper() (*programslicer.ProgramWrapper, error) {
	var (
		program *programslicer.ProgramWrapper
		err     error
	)
	if instDirectory != "" {
		program, err = programslicer.GetProgramWrapperDirectory(instDirectory)
		if err != nil {
			return program, err
		}
		err = InplaceDirectorySwap(instDirectory)
		if err != nil {
			return program, err
		}
	} else if instFile != "" {
		program, err = programslicer.GetProgramWrapperFile(instFile)
		if err != nil {
			return program, err
		}
	}
	//TODO write functionality for piping
	return program, nil
}
func cleanSources(p *programslicer.ProgramWrapper) {
	for pnum, pack := range p.Packages {
		for snum, source := range pack.Sources {
			ast.Inspect(source.Comments, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ExprStmt:
					switch c := x.X.(type) {
					case *ast.CallExpr:
						switch s := c.Fun.(type) {
						case *ast.SelectorExpr: 
							switch xx := s.X.(type) {
							case *ast.Ident:
							fmt.Println(xx.Name)
							fmt.Println(s.Sel.Name)
								if xx.Name == "dinvRT" && s.Sel.Name == "Dump" {
									x.X = ast.NewIdent("//@dump")
								}
							}
						}
					}
				}
				return true
			})
			removeImports(p.Fset,p.Packages[pnum].Sources[snum].Comments)
			buf := new(bytes.Buffer)
			printer.Fprint(buf, p.Fset, p.Packages[pnum].Sources[snum].Comments)
			p.Packages[pnum].Sources[snum].Text = buf.String()
			fmt.Println(buf.String()) //TODO remove

			writeInstrumentedFile(p, pnum, snum)
		}
	}
	return
}
func InplaceDirectorySwap(dir string) error {
	newDir := dir + "_orig"
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		newPath := strings.Replace(path, dir, newDir, -1)
		logger.Printf("moving %s to %s\n", path, newPath)
		if info.IsDir() {
			return os.Mkdir(newPath, 0775)
		} else {
			return os.Rename(path, newPath)
		}
	})
}

//generateCode constructs code for dump statements for the source code
//located at program.source[snum].
func InsturmentSource(program *programslicer.ProgramWrapper, pnum, snum int) {
	dumpNodes := GetDumpNodes(program.Packages[pnum].Sources[snum].Comments)
	affected := getAffectedVars(program)
	fmt.Printf("affected functions size %d\n",len(affected))

	//check for dumps and write imports here
	fmt.Printf("dumpnodes %d\n",len(dumpNodes))
	if len(dumpNodes) > 0 {
		addImports(program.Fset, program.Packages[pnum].Sources[snum].Comments)
	}
	for _, dump := range dumpNodes {
		lineNumber := program.Fset.Position(dump.Pos()).Line
		collectedVariables := getAccessedAffectedVars(dump,affected,program)
		fmt.Printf("collected variables #%d\n",len(collectedVariables))
		dumpcode := GenerateDumpCode(collectedVariables, lineNumber, dump.Text, program.Packages[pnum].Sources[snum].Filename, program.Packages[pnum].PackageName)
		dump.Text = dumpcode
		logger.Println(dumpcode)


	}
	//write the text of the source code out
	buf := new(bytes.Buffer)
	printer.Fprint(buf, program.Fset, program.Packages[pnum].Sources[snum].Comments)
	program.Packages[pnum].Sources[snum].Text = buf.String()
	fmt.Println(buf.String()) //TODO remove
}

//getAccessedAffectedVars returns the names of all variables affected
//by a send, or a receive within the scope of the dump statement
func getAccessedAffectedVars(dump *ast.Comment, affectedFuncs map[*ast.FuncDecl][]*programslicer.FuncNode,  program *programslicer.ProgramWrapper) []string {
	//check that the node is within the known program
	pnum, snum := program.FindFile(dump)
	if pnum < 0 || snum < 0 {
		fmt.Println("Pacakge or Source does not exist")
		return nil
	}
	_, f := findFunction(dump,program.Packages[pnum].Sources[snum].Source.Decls) //NOTE changed from comments.decls for debugging july 4
	if f == nil {
		fmt.Println("Function Not found")
		return nil
	}
	//find variables within the scope of the dump statement
	var affected []string
	inScope := GetAccessibleVarsInScope(int(dump.Pos()), program.Packages[pnum].Sources[snum].Comments, program.Fset)
	//collect all the variables affected by networking in the known
	//function
	fmt.Printf("affected functions size %d\n",len(affectedFuncs))
	for _, fn := range affectedFuncs[f] {
		names := make([]string,0)
		for _, vars := range fn.NVars {
			names = append(names,vars.Name())
		}
		fmt.Println("names: %s\n",names)
		affected = append(affected,names...)
		affected = append(affected,collectStructs(names,program.Packages[pnum].Sources[snum].Comments)...)
	}
	//remove duplicates
	inScope = removedups(inScope)
	affected = removedups(affected)
	//find variables both in scope and affected
	var affectedInScope []string
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

func removedups (slice []string) []string {
	encountered := make(map[string]bool,len(slice))
	noDups := make([]string,0)
	for _, e := range slice {
		if !encountered[e] {
			encountered[e] = true
			noDups = append(noDups,e)
		}
	}
	return noDups
}

//findFunction searches through a set of declaractions decls, for the
//statement stmt, the number of the function, which contains the stmt
//is returned
func findFunction(n ast.Node, decls []ast.Decl) (int, *ast.FuncDecl) {
	fcount := -1
	for dcl := 0; dcl < len(decls)-1; dcl++ {
		_ , ok := decls[dcl].(*ast.FuncDecl)
		if ok {
			fcount++
		}
		if n.Pos() > decls[dcl].Pos() && n.Pos() < decls[dcl+1].Pos() {
			return fcount, decls[dcl].(*ast.FuncDecl)
		}
	}
	if n.Pos() > decls[len(decls)-1].Pos(){
		return fcount+1, decls[len(decls)-1].(*ast.FuncDecl)
	}
	return -1, nil
}
//getAfffectedVars searches through an entire program specified by
//program, and returns the names of all variables modified by
//interprocess communication.

//TODO getAffectedVars does not work at the moment and should be
//restructured. The variables returned should be thoses affected by
//IPC around a particular dump statement, not the entire program
func getAffectedVars(program *programslicer.ProgramWrapper) map[*ast.FuncDecl][]*programslicer.FuncNode {
	sending, receiving, both := capture.GetCommNodes(program)
	affectedFunctions := make(map[*ast.FuncDecl][]*programslicer.FuncNode)
	for _, send := range sending {
		fmt.Printf("Slicing from sender %s\n",send)
		sendStmt := (*send).(ast.Stmt)
		funcNodes := programslicer.GetAffectedVariables(sendStmt,program,programslicer.ComputeBackwardSlice,programslicer.GetTaintedPointsBackwards)
		for f , fNode := range funcNodes {
			affectedFunctions[f] = append(affectedFunctions[f],fNode)
		}
	}
	for _, rec := range receiving {
		fmt.Printf("Slicing from recv %s\n",rec)
		recStmt := (*rec).(ast.Stmt)
		funcNodes := programslicer.GetAffectedVariables(recStmt,program,programslicer.ComputeForwardSlice,programslicer.GetTaintedPointsForward)
		for f , fNode := range funcNodes {
			affectedFunctions[f] = append(affectedFunctions[f],fNode)
		}
	}
	for _, bidir := range both {
		fmt.Printf("Slicing from bidir %s\n",bidir)
		commStmt := (*bidir).(ast.Stmt)
		forwards := programslicer.GetAffectedVariables(commStmt,program,programslicer.ComputeForwardSlice,programslicer.GetTaintedPointsForward)
		for f , fNode := range forwards {
			affectedFunctions[f] = append(affectedFunctions[f],fNode)
		}
		backwards := programslicer.GetAffectedVariables(commStmt,program,programslicer.ComputeBackwardSlice,programslicer.GetTaintedPointsBackwards)
		for f , fNode := range backwards {
			affectedFunctions[f] = append(affectedFunctions[f],fNode)
		}
	}
	return affectedFunctions
}

//GetAccessibleVarsInScope returns the variables names of all
//varialbes in scope at the point start.
func GetAccessibleVarsInScope(dumpPosition int, file *ast.File, fset *token.FileSet) []string {
	logger.Println("Collecting Scope Variables")
	globals := GetGlobalVariables(file, fset)
	locals := GetLocalVariables(dumpPosition, file, fset)
	return append(globals, locals...)
}

func GetGlobalVariables(file *ast.File, fset *token.FileSet) []string {
	var results []string

	global_objs := file.Scope.Objects
	for identifier, _ := range global_objs {
		//get variables of type constant and Var
		switch global_objs[identifier].Kind {
		case ast.Var, ast.Con: //|| global_objs[identifier].Kind == ast.Typ { //can be used for diving into structs
			fmt.Printf("Global Found :%s\n", fmt.Sprintf("%v", identifier))
			results = append(results, fmt.Sprintf("%v", identifier))
		}
	}
	structVars := collectStructs(results, file)
	results = append(results, structVars...)
	return results
}

func GetLocalVariables(dumpPosition int, file *ast.File, fset *token.FileSet) []string {
	var results []string
	filePos := fset.File(file.Package)
	logger.Printf("packagename : %s\n searching Pos dumpPosition %d\n", file.Name.String(), dumpPosition)
	//TODO rename path and write comments
	//the +2 is probably to grab a send or receive after the dump??
	//if the dump is outside of the file return nothing
	if dumpPosition > filePos.Size() || dumpPosition+2 > filePos.Size() {
		return make([]string, 0)
	}
	path, _ := astutil.PathEnclosingInterval(file, filePos.Pos(dumpPosition), filePos.Pos(dumpPosition+2)) // why +2
	//collect the parameters to the function
	if len(path) > 0 {
		_, f := findFunction(path[0], file.Decls)
		if f != nil {
			for _, feilds := range f.Type.Params.List {
				for _, param := range feilds.Names {
					results = append(results, param.Name)
				}
			}
		}
	}
	for _, astnode := range path {
		logger.Println("%v", astutil.NodeDescription(astnode))
		switch t := astnode.(type) {
		case *ast.BlockStmt:
			stmts := t.List
			logger.Printf("Block found at position :%d of size %d\n", int(t.Pos()), len(stmts))
			for _, stmtnode := range stmts {
				logger.Printf("Statement type:%s", stmtnode)
				switch t := stmtnode.(type) {
				case *ast.DeclStmt:
					idents := t.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names
					for _, identifier := range idents {
						//collect node if in scope at the dump statement
						if int(identifier.Pos()) < dumpPosition && identifier.Name != "_" {
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
									if int(resolvedNode.Pos()) < dumpPosition && resolvedNode.Name != "_" {
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
	//Compute the closure of structures in the variables
	structVars := collectStructs(results, file)
	results = append(results, structVars...)
	fmt.Printf("local Vars: %s\n",results)
	return results
}


type structIds struct {
	fields []string
	types  []string
}


func collectStructs(varNames []string, file *ast.File) []string {
	var structs map[string]structIds = make(map[string]structIds)
	//Collect all the structs by name field and type in the program
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			switch typ := x.Type.(type) {
			case *ast.StructType:
				tmp := new(structIds)
				name := x.Name.Name
				for _, field := range typ.Fields.List {
					if len(field.Names) < 1 {
						return false
					}
					tmp.fields = append(tmp.fields, field.Names[0].Name)
					tmpType, ok := field.Type.(*ast.Ident)
					if !ok {
						return false
					}
					tmp.types = append(tmp.types, tmpType.Name)

				}
				structs[name] = *tmp
			}
		}
		return true
	})
	//add the named extensions to each struce itterativle
	var structResults []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ValueSpec:
			for i := range varNames {
				if x.Names[0].Name == varNames[i] {
					structType, ok := x.Type.(*ast.Ident)
					if !ok {
						return false
					}
					_, ok = structs[structType.Name]
					if ok {
						structResults = append(structResults, structClosure(structs, x.Names[0].Name, structType.Name)...)
					}

				}
			}
		}
		return true
	})
	return structResults
}

//structClosure returns the names of all struct varialbes, including
//nested structs
func structClosure(s map[string]structIds, name, stype string) []string {
	var names []string
	id := s[stype]
	for i := range id.fields {
		names = append(names, name+"."+id.fields[i])
		_, ok := s[id.types[i]]
		if ok {
			names = append(names, structClosure(s, name+"."+id.fields[i], id.types[i])...)
		}
	}
	return names

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
func GenerateDumpCode(vars []string, lineNumber int, comment, path, packagename string) string {
	if len(vars) <= 0 {
		return "//@dump (This line [" + strconv.Itoa(lineNumber) + "] contains no in-scope networking variables)"
	}
	_, nameWithExt := filepath.Split(path)
	ext := filepath.Ext(path)
	filename := strings.Replace(nameWithExt, ext, "", 1)
	var buffer bytes.Buffer

	// write vars' values
	id := packagename + "_" + filename + "_" + strconv.Itoa(lineNumber) + "_"
	var varlist string
	var namedVarList string
	for i:=0;i<(len(vars)-1);i++ {
		namedVarList += id + vars[i] + ","
		varlist += vars[i] + ","
	}
	namedVarList += id + vars[len(vars)-1]
	varlist+= vars[len(vars)-1]
	dumpString := "dinvRT.Dump(\"" + namedVarList + "\"," + varlist + ")"

	buffer.WriteString(dumpString)
	return buffer.String()
}

//prints given AST
func printAST(p *programslicer.ProgramWrapper) {
	for _, pack := range p.Packages {
		for _, source := range pack.Sources {
			ast.Print(p.Fset, source.Source)
		}
	}
}

func writeInstrumentedFile (p *programslicer.ProgramWrapper, pnum, snum int) {
	filename := p.Packages[pnum].Sources[snum].Filename
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	logger.Printf("Writing file %s\n",filename)
	file.WriteString(p.Packages[pnum].Sources[snum].Text)
	file.Close()
}

func addImports(fset *token.FileSet, file *ast.File) {
	packagesToImport := []string{"bitbucket.org/bestchai/dinv/dinvRT"}
	for _, pack := range packagesToImport {
		astutil.AddImport(fset,file,pack)
	}
}

func removeImports(fset *token.FileSet, file * ast.File) {
	packagesToRemove := []string{"bitbucket.org/bestchai/dinv/dinvRT"}
	for _, pack := range packagesToRemove {
		if !astutil.UsesImport(file,pack) {
			astutil.DeleteImport(fset,file,pack)
		}
	}
	return
}

