//ProgramBuilder.go constructs a wrapper for the package of code being
//instrumented. The wrapper is built in tiers, with the
//ProgramWrapper representing an entire package, SourceWrapper
//defining one source code file, and CFGWrapper representing a control
//flow graph for a function

package instrumenter

import (
	"go/ast"
	"go/parser"
	"go/token"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
	"golang.org/x/tools/go/loader"
	"go/types"
)

//Program wrapper is a wrapper for an entire package, the source code
//of every file in the package is found in source.
//TODO make source -> sources
type ProgramWrapper struct {
	prog     *loader.Program
	fset     *token.FileSet
	packages []*PackageWrapper
}

type PackageWrapper struct {
	packageName string
	sources     []*SourceWrapper
}

//SourceWrapper abstracts a single source file. Text is the string
//represtation of the source file. Each source wrapper contains a CFG
//for each function defined.
type SourceWrapper struct {
	comments *ast.File
	source   *ast.File
	filename string
	text     string
	cfgs     []*CFGWrapper
}

//CFGWrapper abstract a control flow graph for a single function, The
//statements and objects in the function are made available for
//convienence.
type CFGWrapper struct {
	cfg      *cfg.CFG
	exp      map[int]ast.Stmt
	stmts    map[ast.Stmt]int
	objs     map[string]*types.Var
	objNames map[*types.Var]string
}

func LoadProgram(sourceFiles []*ast.File, config loader.Config) (*loader.Program, error) {
	//logger.Println("Loading Packages")
	config.CreateFromFiles("testing", sourceFiles...)
	prog, err := config.Load()
	if err != nil {
		return nil, err
	}
	return prog, nil
}

func getWrapperFromString(sourceString string) (*ProgramWrapper, error) {
	var config loader.Config
	fset := token.NewFileSet()
	comments, err := parser.ParseFile(fset, "single", sourceString, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	source, err := config.ParseFile("single", sourceString)
	if err != nil {
		return nil, err
	}
	filename := comments.Name.String()
	//make the single files the head of a list
	sources := append(make([]*ast.File, 0), source)
	prog, err := LoadProgram(sources, config)
	if err != nil {
		return nil, err
	}
	pack := genPackageWrapper(
		append(make([]*ast.File, 0), source),
		append(make([]*ast.File, 0), comments),
		append(make([]string, 0), filename),
		fset,
		prog)
	return &ProgramWrapper{prog, fset, append(make([]*PackageWrapper, 0), pack)}, nil
}

func getProgramWrapperDirectory(dir string) (*ProgramWrapper, error) {
	var config loader.Config
	fset := token.NewFileSet()
	astPackages, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	packageWrappers := make([]*PackageWrapper, 0)
	sourceFiles := make(map[string][]*ast.File, len(astPackages))
	commentFiles := make(map[string][]*ast.File, len(astPackages))
	filenames := make(map[string][]string, len(astPackages))
	for i, Package := range astPackages {
		sourceFiles[i], commentFiles[i], filenames[i] = getSourceAndCommentFiles(Package, config)
	}
	aggergateSources := make([]*ast.File, 0)
	for i := range sourceFiles {
		aggergateSources = append(aggergateSources, sourceFiles[i]...)
	}
	prog, err := LoadProgram(aggergateSources, config)
	if err != nil {
		return nil, err
	}
	for packageIndex := range sourceFiles {
		packageWrappers = append(packageWrappers,
			genPackageWrapper(sourceFiles[packageIndex],
				commentFiles[packageIndex],
				filenames[packageIndex],
				fset,
				prog))
	}
	return &ProgramWrapper{prog, fset, packageWrappers}, nil
}

//getSourceAndCommentFiles scrapes all of the source files in the
//package (packageName) ast's of each of the source files are built, a
//corresponding ast of the comments is also built for dump statement
//analysis
func getSourceAndCommentFiles(astPackage *ast.Package, config loader.Config) (sources, comments []*ast.File, filenames []string) {

	sources = make([]*ast.File, 0)
	comments = make([]*ast.File, 0)
	filenames = make([]string, 0)
	for name := range astPackage.Files {
		source, err := config.ParseFile(name, nil)
		if err != nil {
			return nil, nil, nil
		}
		sources = append(sources, source)
		comments = append(comments, astPackage.Files[name])
		filenames = append(filenames, name)
		logger.Println(name)
	}
	return sources, comments, filenames
}

func genPackageWrapper(sourceFiles []*ast.File, commentFiles []*ast.File, filenames []string, fset *token.FileSet, prog *loader.Program) *PackageWrapper {
	pName := commentFiles[0].Name.String()
	sources := make([]*SourceWrapper, 0)
	logger.Println(len(filenames), len(sourceFiles))
	for i, file := range sourceFiles {
		logger.Printf("building source for %s\n", filenames[i])
		cfgs := make([]*CFGWrapper, 0)
		for j := 0; j < len(file.Decls); j++ {
			logger.Printf("building CFG[%d]\n", j)
			functionDec, ok := file.Decls[j].(*ast.FuncDecl)
			if ok {
				logger.Printf("FuncFound\n")
				wrap := getWrapper(functionDec, prog)
				cfgs = append(cfgs, wrap)
			}
		}
		logger.Println("Source Built")
		sources = append(sources, &SourceWrapper{
			comments: commentFiles[i],
			source:   sourceFiles[i],
			filename: filenames[i],
			cfgs:     cfgs})
	}
	logger.Println("Wrappers Built")
	return &PackageWrapper{
		packageName: pName,
		sources:     sources,
	}
}

const (
	START = 0
	END   = 100000000
)

//getWrapper creates a wrapper for a control flow graph
func getWrapper(functionDec *ast.FuncDecl, prog *loader.Program) *CFGWrapper {
	cfg := cfg.FromFunc(functionDec)
	v := make(map[int]ast.Stmt)
	stmts := make(map[ast.Stmt]int)
	objs := make(map[string]*types.Var)
	objNames := make(map[*types.Var]string)
	i := 1
	//logger.Println("GETTING WRAPPER")
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
