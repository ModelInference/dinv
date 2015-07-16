// Copyright 2015 Auburn University. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cfg provides intraprocedural control flow graphs (CFGs) with
// statement-level granularity, i.e., CFGs whose nodes correspond 1-1 to the
// Stmt nodes from an abstract syntax tree.
package cfg

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// This package can be used to construct a control flow graph from an abstract syntax tree (go/ast).
// This is done by traversing a list of statements (likely from a Block)
// depth-first and creating an adjacency list, implemented as a map of Blocks.
// Adjacent Blocks are stored as predecessors and successors separately for
// control flow information. Any defers encountered while traversing the ast
// will be added to a slice that can be accessed from CFG. Their behavior is such
// that they may or may not be flowed to, potentially multiple times, after Exit.
// This behavior is dependant upon in what control structure they were found,
// i.e. if/for body may never be flowed to.

// TODO(you): defers are lazily done currently. If needed, could likely use a more robust
//  implementation wherein they are represented as a graph after Exit.
// TODO(reed): closures, go func() ?

// CFG defines a control flow graph with statement-level granularity, in which
// there is a 1-1 correspondence between a Block in the CFG and an ast.Stmt.
type CFG struct {
	// Sentinel nodes for single-entry, single-exit CFG. Not in original AST.
	Entry, Exit *ast.BadStmt
	// All defers found in CFG, disjoint from Blocks. May be flowed to after Exit.
	Defers     []*ast.DeferStmt
	Blocks     map[ast.Stmt]*Block //TODO add wrappr
	BlockSlice []*Block            //TODO // just an array of every thing in Blocks added move to own library ( program slice )
}

type Block struct {
	Index        int //TODO remove added
	Stmt         ast.Stmt
	Preds        []ast.Stmt
	Succs        []ast.Stmt
	PredBs       []*Block //TODO from here down all fields are added
	SuccBs       []*Block
	dom          domInfo
	ControlDep   []*Block
	ControlDepee []*Block
	DataDep      []*Block
	DataDepee    []*Block
}

//TODO move into own library (control dep)
func (c *CFG) InitializeBlocks() {
	//define preceding and successive block statements based on
	//pres/succ
	for _, b := range c.Blocks {
		for _, stmt := range b.Preds {
			b.PredBs = append(b.PredBs, c.Blocks[stmt])

		}
		for _, stmt := range b.Succs {
			b.SuccBs = append(b.SuccBs, c.Blocks[stmt])
		}
	}
	c.BlockSlice = make([]*Block, 0)
	for _, b := range c.Blocks {
		c.BlockSlice = append(c.BlockSlice, b)

	}

	for i, b := range c.BlockSlice {
		if b.Stmt == c.Entry {
			c.BlockSlice[0], c.BlockSlice[i] = c.BlockSlice[i], c.BlockSlice[0]
			break
		}
	}

	for i, b := range c.BlockSlice {
		b.Index = i
	}

}

//TODO move into own library (control dep)
func (c *CFG) InvertCFG() *CFG {
	invCFG := new(CFG)

	invCFG.Entry = c.Exit
	invCFG.Exit = c.Entry
	invCFG.Blocks = make(map[ast.Stmt]*Block)
	for _, b := range c.Blocks {
		invB := new(Block)
		invB.Stmt = b.Stmt
		invB.Preds = b.Succs
		invB.Succs = b.Preds
		invB.DataDep = b.DataDep
		invB.DataDepee = b.DataDepee
		invCFG.Blocks[invB.Stmt] = invB
	}
	invCFG.InitializeBlocks()

	return invCFG

}

// FromStmts returns the control-flow graph for the given sequence of statements.
func FromStmts(s []ast.Stmt) *CFG {
	return newBuilder().build(s)
}

// FromFunc is a convenience function for creating a CFG from a given function declaration.
func FromFunc(f *ast.FuncDecl) *CFG {
	return FromStmts(f.Body.List)
}

// Preds returns a slice of all immediate predecessors for the given statement.
// May include Entry node.
func (c *CFG) Preds(s ast.Stmt) []ast.Stmt {
	return c.Blocks[s].Preds
}

// Succs returns a slice of all immediate successors to the given statement.
// May include Exit node.
func (c *CFG) Succs(s ast.Stmt) []ast.Stmt {
	return c.Blocks[s].Succs
}

// Blocks returns a slice of all Blocks in a CFG, including the Entry and Exit nodes.
// The Blocks are in no particular order.
func (c *CFG) GetBlocks() []ast.Stmt {
	Blocks := make([]ast.Stmt, 0, len(c.Blocks))
	for s, _ := range c.Blocks {
		Blocks = append(Blocks, s)
	}
	return Blocks
}

func (c *CFG) PrintDot(f io.Writer, fset *token.FileSet, addl func(n ast.Stmt) string) {
	fmt.Fprintf(f, `digraph mgraph {
mode="heir";
splines="ortho";

`)
	for _, v := range c.Blocks {
		for _, a := range v.Succs {
			fmt.Fprintf(f, "\t\"%s\" -> \"%s\"\n",
				c.printVertex(v, fset, addl(v.Stmt)),
				c.printVertex(c.Blocks[a], fset, addl(c.Blocks[a].Stmt)))
		}
	}
	fmt.Fprintf(f, "}\n")
}

//TODO move into own library ( control Dep )
func (c *CFG) PrintControlDepDot(f io.Writer, fset *token.FileSet, addl func(n ast.Stmt) string) {
	fmt.Fprintf(f, `digraph mgraph {
mode="heir";
splines="ortho";

`)
	for _, v := range c.Blocks {
		for _, a := range v.ControlDep {
			fmt.Fprintf(f, "\t\"%s\" -> \"%s\"\n",
				c.printVertex(v, fset, addl(v.Stmt)),
				c.printVertex(a, fset, addl(a.Stmt)))
		}
	}
	fmt.Fprintf(f, "}\n")
}

//TODO move to own library ( control Dep )
func (c *CFG) PrintDataDepDot(f io.Writer, fset *token.FileSet, addl func(n ast.Stmt) string) {
	fmt.Fprintf(f, `digraph mgraph {
mode="heir";
splines="ortho";

`)
	for _, v := range c.Blocks {
		for _, a := range v.DataDep {
			fmt.Fprintf(f, "\t\"%s\" -> \"%s\"\n",
				c.printVertex(v, fset, addl(v.Stmt)),
				c.printVertex(a, fset, addl(a.Stmt)))
		}
	}
	fmt.Fprintf(f, "}\n")
}

func (c *CFG) printVertex(v *Block, fset *token.FileSet, addl string) string {
	switch v.Stmt {
	case c.Entry:
		return "ENTRY"
	case c.Exit:
		return "EXIT"
	case nil:
		return ""
	}
	addl = strings.Replace(addl, "\n", "\\n", -1)
	if addl != "" {
		addl = "\\n" + addl
	}
	return fmt.Sprintf("%s - line %d%s",
		astutil.NodeDescription(v.Stmt),
		fset.Position(v.Stmt.Pos()).Line,
		addl)
}

type builder struct {
	Blocks      map[ast.Stmt]*Block
	prev        []ast.Stmt        // Blocks to hook up to current Block
	branches    []*ast.BranchStmt // accumulated branches from current inner Blocks
	entry, exit *ast.BadStmt      // single-entry, single-exit nodes
	defers      []*ast.DeferStmt  // all defers encountered
}

func newBuilder() *builder {
	return &builder{
		Blocks: make(map[ast.Stmt]*Block),
		entry:  new(ast.BadStmt),
		exit:   new(ast.BadStmt),
	}
}

// build runs buildBlock on the given Block (traversing nested statements), and
// adds entry and exit nodes.
func (b *builder) build(s []ast.Stmt) *CFG {
	b.prev = []ast.Stmt{b.entry}
	b.buildBlock(s)
	b.addSucc(b.exit)

	return &CFG{
		Blocks: b.Blocks,
		Entry:  b.entry,
		Exit:   b.exit,
		Defers: b.defers,
	}
}

// addSucc adds a control flow edge from all previous Blocks to the Block for
// the given statement.
func (b *builder) addSucc(current ast.Stmt) {
	cur := b.Block(current)

	for _, p := range b.prev {
		p := b.Block(p)
		p.Succs = appendNoDuplicates(p.Succs, cur.Stmt)
		cur.Preds = appendNoDuplicates(cur.Preds, p.Stmt)
	}
}

func appendNoDuplicates(list []ast.Stmt, Stmt ast.Stmt) []ast.Stmt {
	for _, s := range list {
		if s == Stmt {
			return list
		}
	}
	return append(list, Stmt)
}

// Block returns a Block for the given statement, creating one and inserting it
// into the CFG if it doesn't already exist.
func (b *builder) Block(s ast.Stmt) *Block {
	bl, ok := b.Blocks[s]
	if !ok {
		bl = &Block{Stmt: s}
		b.Blocks[s] = bl
	}
	return bl
}

// buildStmt adds the given statement and all nested statements to the control
// flow graph under construction. Upon completion, b.prev is set to all
// control flow exits generated from traversing cur.
func (b *builder) buildStmt(cur ast.Stmt) {
	if dfr, ok := cur.(*ast.DeferStmt); ok {
		b.defers = append(b.defers, dfr)
		return // never flow to or from defer
	}

	// Each buildXxx method will flow the previous Blocks to itself appropiately and also
	// set the appropriate Blocks to flow from at the end of the method.
	switch cur := cur.(type) {
	case *ast.BlockStmt:
		b.buildBlock(cur.List)
	case *ast.IfStmt:
		b.buildIf(cur)
	case *ast.ForStmt, *ast.RangeStmt:
		b.buildLoop(cur)
	case *ast.SwitchStmt, *ast.SelectStmt, *ast.TypeSwitchStmt:
		b.buildSwitch(cur)
	case *ast.BranchStmt:
		b.buildBranch(cur)
	case *ast.LabeledStmt:
		b.addSucc(cur)
		b.prev = []ast.Stmt{cur}
		b.buildStmt(cur.Stmt)
	case *ast.ReturnStmt:
		b.addSucc(cur)
		b.prev = []ast.Stmt{cur}
		b.addSucc(b.exit)
		b.prev = nil
	default: // most statements have straight-line control flow
		b.addSucc(cur)
		b.prev = []ast.Stmt{cur}
	}
}

func (b *builder) buildBranch(br *ast.BranchStmt) {
	b.addSucc(br)
	b.prev = []ast.Stmt{br}

	switch br.Tok {
	case token.FALLTHROUGH:
		// successors handled in buildSwitch, so skip this here
	case token.GOTO:
		b.addSucc(br.Label.Obj.Decl.(ast.Stmt)) // flow to label
	case token.BREAK, token.CONTINUE:
		b.branches = append(b.branches, br) // to handle at switch/for/etc level
	}
	b.prev = nil // successors handled elsewhere
}

func (b *builder) buildIf(f *ast.IfStmt) {
	if f.Init != nil {
		b.addSucc(f.Init)
		b.prev = []ast.Stmt{f.Init}
	}
	b.addSucc(f)

	b.prev = []ast.Stmt{f}
	b.buildBlock(f.Body.List) // build then

	ctrlExits := b.prev // aggregate of b.prev from each condition

	switch s := f.Else.(type) {
	case *ast.BlockStmt: // build else
		b.prev = []ast.Stmt{f}
		b.buildBlock(s.List)
		ctrlExits = append(ctrlExits, b.prev...)
	case *ast.IfStmt: // build else if
		b.prev = []ast.Stmt{f}
		b.addSucc(s)
		b.buildIf(s)
		ctrlExits = append(ctrlExits, b.prev...)
	case nil: // no else
		ctrlExits = append(ctrlExits, f)
	}

	b.prev = ctrlExits
}

// buildLoop builds CFG Blocks for a ForStmt or RangeStmt, including nested statements.
// Upon return, b.prev set to for and any appropriate breaks.
func (b *builder) buildLoop(Stmt ast.Stmt) {
	// flows as such (range same w/o init & post):
	// previous -> [ init -> ] for -> body -> [ post -> ] for -> next

	var post ast.Stmt = Stmt // post in for loop, or for Stmt itself; body flows to this

	switch Stmt := Stmt.(type) {
	case *ast.ForStmt:
		if Stmt.Init != nil {
			b.addSucc(Stmt.Init)
			b.prev = []ast.Stmt{Stmt.Init}
		}
		b.addSucc(Stmt)

		if Stmt.Post != nil {
			post = Stmt.Post
			b.prev = []ast.Stmt{post}
			b.addSucc(Stmt)
		}

		b.prev = []ast.Stmt{Stmt}
		b.buildBlock(Stmt.Body.List)
	case *ast.RangeStmt:
		b.addSucc(Stmt)
		b.prev = []ast.Stmt{Stmt}
		b.buildBlock(Stmt.Body.List)
	}

	b.addSucc(post)

	ctrlExits := []ast.Stmt{Stmt}

	// handle any branches; if no label or for me: handle and remove from branches.
	for i := 0; i < len(b.branches); i++ {
		br := b.branches[i]
		if br.Label == nil || br.Label.Obj.Decl.(*ast.LabeledStmt).Stmt == Stmt {
			switch br.Tok { // can only be one of these two cases
			case token.CONTINUE:
				b.prev = []ast.Stmt{br}
				b.addSucc(post) // connect to .Post statement if present, for Stmt otherwise
			case token.BREAK:
				ctrlExits = append(ctrlExits, br)
			}
			b.branches = append(b.branches[:i], b.branches[i+1:]...)
			i-- // removed in place, so go back to this i
		}
	}

	b.prev = ctrlExits // for Stmt and any appropriate break statements
}

// buildSwitch builds a multi-way branch statement, i.e. switch, type switch or select.
// Upon return, each case's control exits set as b.prev.
func (b *builder) buildSwitch(sw ast.Stmt) {
	// composition of statement sw:
	//
	//    sw: *ast.SwitchStmt || *ast.TypeSwitchStmt || *ast.SelectStmt
	//      Body.List: []*ast.CaseClause || []ast.CommClause
	//        clause: []ast.Stmt

	var cases []ast.Stmt // case 1:, case 2:, ...

	switch sw := sw.(type) {
	case *ast.SwitchStmt: // i.e. switch [ x := 0; ] [ x ] { }
		if sw.Init != nil {
			b.addSucc(sw.Init)
			b.prev = []ast.Stmt{sw.Init}
		}
		b.addSucc(sw)
		b.prev = []ast.Stmt{sw}

		cases = sw.Body.List
	case *ast.TypeSwitchStmt: // i.e. switch [ x := 0; ] t := x.(type) { }
		if sw.Init != nil {
			b.addSucc(sw.Init)
			b.prev = []ast.Stmt{sw.Init}
		}
		b.addSucc(sw)
		b.prev = []ast.Stmt{sw}
		b.addSucc(sw.Assign)
		b.prev = []ast.Stmt{sw.Assign}

		cases = sw.Body.List
	case *ast.SelectStmt: // i.e. select { }
		b.addSucc(sw)
		b.prev = []ast.Stmt{sw}

		cases = sw.Body.List
	}

	var caseExits []ast.Stmt // aggregate of b.prev's resulting from each case
	swPrev := b.prev         // save for each case's previous; Switch or Assign
	var ft *ast.BranchStmt   // fallthrough to handle from previous case, if any
	defaultCase := false

	for _, clause := range cases {
		b.prev = swPrev
		b.addSucc(clause)
		b.prev = []ast.Stmt{clause}
		if ft != nil {
			b.prev = append(b.prev, ft)
		}

		var caseBody []ast.Stmt

		// both of the following cases are guaranteed in spec
		switch clause := clause.(type) {
		case *ast.CaseClause: // i.e. case: [expr,expr,...]:
			if clause.List == nil {
				defaultCase = true
			}
			caseBody = clause.Body
		case *ast.CommClause: // i.e. case c <- chan:
			if clause.Comm == nil {
				defaultCase = true
			} else {
				b.addSucc(clause.Comm)
				b.prev = []ast.Stmt{clause.Comm}
			}
			caseBody = clause.Body
		}

		b.buildBlock(caseBody)

		if ft = fallThrough(caseBody); ft == nil {
			caseExits = append(caseExits, b.prev...)
		}
	}

	if !defaultCase {
		caseExits = append(caseExits, swPrev...)
	}

	// handle any breaks that are unlabeled or for me
	for i := 0; i < len(b.branches); i++ {
		br := b.branches[i]
		if br.Tok == token.BREAK && (br.Label == nil || br.Label.Obj.Decl.(*ast.LabeledStmt).Stmt == sw) {
			caseExits = append(caseExits, br)
			b.branches = append(b.branches[:i], b.branches[i+1:]...)
			i-- // we removed in place, so go back to this index
		}
	}

	b.prev = caseExits // control exits of each case and breaks
}

// fallThrough returns the fallthrough Stmt at the end of stmts, if one exists,
// and nil otherwise.
func fallThrough(stmts []ast.Stmt) *ast.BranchStmt {
	if len(stmts) < 1 {
		return nil
	}

	// fallthrough can only be last statement in clause (possibly labeled)
	ft := stmts[len(stmts)-1]

	for { // recursively descend LabeledStmts.
		switch s := ft.(type) {
		case *ast.BranchStmt:
			if s.Tok == token.FALLTHROUGH {
				return s
			}
		case *ast.LabeledStmt:
			ft = s.Stmt
			continue
		}
		break
	}
	return nil
}

// buildBlock iterates over a slice of statements (typically the statements
// from an ast.BlockStmt), adding them successively to the CFG.  Upon return,
// b.prev is set to the control exits of the last statement.
func (b *builder) buildBlock(Block []ast.Stmt) {
	for _, Stmt := range Block {
		b.buildStmt(Stmt)
	}
}
