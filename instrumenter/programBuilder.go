package instrumenter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"
)

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

func LoadPackage(sourceFiles []*ast.File, config loader.Config) *loader.Program {
	fmt.Println("Loading Packages")
	config.CreateFromFiles("testing", sourceFiles...)
	prog, err := config.Load()
	if err != nil {
		fmt.Println("CannotLoad")
		return nil
	}
	fmt.Println("Files Loaded")
	return prog
}

func getSourceAndCommentFiles(dir, packageName string, config loader.Config) (sources, comments []*ast.File, filenames []string) {
	astPackages, err := parser.ParseDir(token.NewFileSet(), dir, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil
	}
	astPackage := astPackages[packageName]

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
		fmt.Println(name)
	}
	return sources, comments, filenames
}

func getWrappers(dir, packageName string) *ProgramWrapper {
	var config loader.Config
	sourceFiles, commentFiles, filenames := getSourceAndCommentFiles(dir, packageName, config)
	prog := LoadPackage(sourceFiles, config)
	pName := commentFiles[0].Name.String()

	sources := make([]*SourceWrapper, 0)
	fmt.Println(len(filenames), len(sourceFiles))
	for i, file := range sourceFiles {
		fmt.Printf("building source for %s\n", filenames[i])
		cfgs := make([]*CFGWrapper, 0)
		for j := 0; j < len(file.Decls); j++ {
			fmt.Printf("building CFG[%d]\n", j)
			functionDec, ok := file.Decls[j].(*ast.FuncDecl)
			if ok {
				print("FuncFound\n")
				wrap := getWrapper(functionDec, prog)
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
	return nil
}

//getWrapper creates a wrapper for a control flow graph
func getWrapper(functionDec *ast.FuncDecl, prog *loader.Program) *CFGWrapper {
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
