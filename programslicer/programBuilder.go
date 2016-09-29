//ProgramBuilder.go constructs a wrapper for the package of code being
//instrumented. The wrapper is built in tiers, with the
//ProgramWrapper representing an entire package, SourceWrapper
//defining one Source code file, and CFGWrapper representing a control
//flow graph for a function

package programslicer

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"go/printer"

	"bitbucket.org/bestchai/dinv/programslicer/cfg"
	"go/types"
	"golang.org/x/tools/go/loader"
)

//Program wrapper is a wrapper for an entire package, the Source code
//of every file in the package is found in Source.
//TODO make Source -> Sources
type ProgramWrapper struct {
	Prog     *loader.Program
	Fset     *token.FileSet
	Packages []*PackageWrapper
}

type PackageWrapper struct {
	PackageName string
	Sources     []*SourceWrapper
}

//SourceWrapper abstracts a single Source file. Text is the string
//represtation of the Source file. Each Source wrapper contains a CFG
//for each function defined.
type SourceWrapper struct {
	Comments *ast.File
	Source   *ast.File
	Filename string
	Text     string
	Cfgs     []*CFGWrapper
}

//CFGWrapper abstract a control flow graph for a single function, The
//statements and objects in the function are made available for
//convienence.
type CFGWrapper struct {
	Cfg      *cfg.CFG
	Exp      map[int]ast.Stmt
	Stmts    map[ast.Stmt]int
	Objs     map[string]*types.Var
	ObjNames map[*types.Var]string
}

func LoadProgram(SourceFiles []*ast.File, config loader.Config) (*loader.Program, error) {
	//fmt.Println("Loading Packages")
	config.CreateFromFiles("testing", SourceFiles...)
	Prog, err := config.Load()
	if err != nil {
		return nil, err
	}
	return Prog, nil
}

func GetWrapperFromString(SourceString string) (*ProgramWrapper, error) {
	var config loader.Config
	Fset := token.NewFileSet()
	Comments, err := parser.ParseFile(Fset, "single", SourceString, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	//ast.Print(Fset,Comments);
	Source, err := config.ParseFile("single", SourceString)
	if err != nil {
		return nil, err
	}
	Filename := Comments.Name.String()
	//make the single files the head of a list
	Sources := append(make([]*ast.File, 0), Source)
	Prog, err := LoadProgram(Sources, config)
	if err != nil {
		return nil, err
	}
	pack := genPackageWrapper(
		append(make([]*ast.File, 0), Source),
		append(make([]*ast.File, 0), Comments),
		append(make([]string, 0), Filename),
		Fset,
		Prog)
	return &ProgramWrapper{Prog, Fset, append(make([]*PackageWrapper, 0), pack)}, nil
}

func GetProgramWrapperFile(path string) (*ProgramWrapper, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var Source string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		Source = Source + scanner.Text() + "\n"
	}
	p, err := GetWrapperFromString(Source)
	if err != nil {
		return nil, err
	}
	p.Packages[0].Sources[0].Filename = path
	return p, nil
}

func GetProgramWrapperDirectory(dir string) (*ProgramWrapper, error) {
	var config loader.Config
	Fset := token.NewFileSet()
	astPackages, err := parser.ParseDir(Fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	packageWrappers := make([]*PackageWrapper, 0)
	SourceFiles := make(map[string][]*ast.File, len(astPackages))
	commentFiles := make(map[string][]*ast.File, len(astPackages))
	Filenames := make(map[string][]string, len(astPackages))
	for i, Package := range astPackages {
		SourceFiles[i], commentFiles[i], Filenames[i] = getSourceAndCommentFiles(Package, config)
	}
	aggergateSources := make([]*ast.File, 0)
	for i := range SourceFiles {
		aggergateSources = append(aggergateSources, SourceFiles[i]...)
	}
	Prog, err := LoadProgram(aggergateSources, config)
	if err != nil {
		return nil, err
	}
	for packageIndex := range SourceFiles {
		packageWrappers = append(packageWrappers,
			genPackageWrapper(SourceFiles[packageIndex],
				commentFiles[packageIndex],
				Filenames[packageIndex],
				Fset,
				Prog))
	}
	return &ProgramWrapper{Prog, Fset, packageWrappers}, nil
}

//refresh rebuilds the program wrapper from it's ast's under the
//this is usefull if they are modified
func (p *ProgramWrapper) Refresh() error {
	var config loader.Config
	fset := token.NewFileSet()

	packageWrappers := make([]*PackageWrapper, 0)
	sourceFiles := make(map[string][]*ast.File, len(p.Packages))
	commentFiles := make(map[string][]*ast.File, len(p.Packages))
	filenames := make(map[string][]string, len(p.Packages))

	for _, pack := range p.Packages {
		for _, source := range pack.Sources {
			//get the source ast
			var buf bytes.Buffer
			printer.Fprint(&buf, fset, source.Source)
			src := buf.String()
			s, _ := parser.ParseFile(fset, source.Filename, src, 0)
			sourceFiles[pack.PackageName] = append(sourceFiles[pack.PackageName], s)
			buf.Reset()
			printer.Fprint(&buf, fset, source.Comments)
			src = buf.String()
			//fmt.Println(src)
			c, _ := parser.ParseFile(fset, source.Filename, src, parser.ParseComments)
			commentFiles[pack.PackageName] = append(commentFiles[pack.PackageName], c)
			filenames[pack.PackageName] = append(filenames[pack.PackageName], source.Filename)
		}
	}
	aggergateSources := make([]*ast.File, 0)
	for i := range sourceFiles {
		aggergateSources = append(aggergateSources, sourceFiles[i]...)
	}
	prog, err := LoadProgram(aggergateSources, config)
	if err != nil {
		return err
	}
	for packageIndex := range sourceFiles {
		packageWrappers = append(packageWrappers,
			genPackageWrapper(sourceFiles[packageIndex],
				commentFiles[packageIndex],
				filenames[packageIndex],
				fset,
				prog))
	}
	p = &ProgramWrapper{prog, fset, packageWrappers}

	return nil
}

//getSourceAndCommentFiles scrapes all of the Source files in the
//package (PackageName) ast's of each of the Source files are built, a
//corresponding ast of the Comments is also built for dump statement
//analysis
func getSourceAndCommentFiles(astPackage *ast.Package, config loader.Config) (Sources, Comments []*ast.File, Filenames []string) {

	Sources = make([]*ast.File, 0)
	Comments = make([]*ast.File, 0)
	Filenames = make([]string, 0)
	for name := range astPackage.Files {
		Source, err := config.ParseFile(name, nil)
		if err != nil {
			return nil, nil, nil
		}
		Sources = append(Sources, Source)
		Comments = append(Comments, astPackage.Files[name])
		Filenames = append(Filenames, name)
	}
	return Sources, Comments, Filenames
}

func genPackageWrapper(SourceFiles []*ast.File, commentFiles []*ast.File, Filenames []string, Fset *token.FileSet, Prog *loader.Program) *PackageWrapper {
	pName := commentFiles[0].Name.String()
	Sources := make([]*SourceWrapper, 0)
	for i, file := range SourceFiles { //NOTE this must be a source file comments will produce irratic analysis (not sure why)
		//fmt.Printf("building Source for %s\n", Filenames[i])
		Cfgs := make([]*CFGWrapper, 0)
		for j := 0; j < len(file.Decls); j++ {
			functionDec, ok := file.Decls[j].(*ast.FuncDecl)
			if ok {
				//fmt.Printf("FuncFound\n")
				//fmt.Printf("building CFG[%d]\n", len(Cfgs))
				wrap := getWrapper(functionDec, Prog)
				if true {
					invC := wrap.Cfg.BuildPostDomTree()
					var buf bytes.Buffer
					invC.PrintDot(&buf, Fset, func(s ast.Stmt) string {
						if _, ok := s.(*ast.AssignStmt); ok {
							return "!"
						} else {
							return ""
						}
					})
					//fmt.Println(buf.String())
					//	fmt.Println(invC.BlockSlice)
				}
				Cfgs = append(Cfgs, wrap)
			}
		}
		if debug {
			fmt.Println("Source Built")
		}
		Sources = append(Sources, &SourceWrapper{
			Comments: commentFiles[i],
			Source:   SourceFiles[i],
			Filename: Filenames[i],
			Cfgs:     Cfgs})
	}
	if debug {
		fmt.Println("Wrappers Built")
	}
	return &PackageWrapper{
		PackageName: pName,
		Sources:     Sources,
	}
}

const (
	START = 0
	END   = 100000000
)

//getWrapper creates a wrapper for a control flow graph
func getWrapper(functionDec *ast.FuncDecl, Prog *loader.Program) *CFGWrapper {
	Cfg := cfg.FromFunc(functionDec)
	v := make(map[int]ast.Stmt)
	Stmts := make(map[ast.Stmt]int)
	Objs := make(map[string]*types.Var)
	ObjNames := make(map[*types.Var]string)
	i := 1
	//fmt.Println("GETTING WRAPPER")
	ast.Inspect(functionDec, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if obj, ok := Prog.Created[0].ObjectOf(x).(*types.Var); ok {
				Objs[obj.Name()] = obj
				ObjNames[obj] = obj.Name()
			}
		case ast.Stmt:
			switch x.(type) {
			case *ast.BlockStmt:
				return true
			}
			v[i] = x
			Stmts[x] = i
			i++
		case *ast.FuncLit:
			// skip statements in anonymous functions
			return false
		}
		return true
	})
	v[END] = Cfg.Exit
	v[START] = Cfg.Entry
	Stmts[Cfg.Entry] = START
	Stmts[Cfg.Exit] = END

	return &CFGWrapper{
		Cfg:      Cfg,
		Exp:      v,
		Stmts:    Stmts,
		Objs:     Objs,
		ObjNames: ObjNames,
	}
}

func (p *ProgramWrapper) FindFile(n ast.Node) (pnum, snum int) {
	for pindex := range p.Packages {
		for sindex := range p.Packages[pindex].Sources {
			if (p.Packages[pindex].Sources[sindex].Comments.Pos() <= n.Pos() &&
				p.Packages[pindex].Sources[sindex].Comments.End() >= n.End()) ||
				(p.Packages[pindex].Sources[sindex].Source.Pos() <= n.Pos() &&
					p.Packages[pindex].Sources[sindex].Source.End() >= n.End()) {
				return pnum, snum
			}
		}
	}
	return -1, -1
}
