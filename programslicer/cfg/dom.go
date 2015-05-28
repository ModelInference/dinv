// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cfg

// This file defines algorithms related to dominance.

// Dominator tree construction ----------------------------------------
//
// We use the algorithm described in Lengauer & Tarjan. 1979.  A fast
// algorithm for finding dominators in a flowgraph.
// http://doi.acm.org/10.1145/357062.357071
//
// We also apply the optimizations to SLT described in Georgiadis et
// al, Finding Dominators in Practice, JGAA 2006,
// http://jgaa.info/accepted/2006/GeorgiadisTarjanWerneck2006.10.1.pdf
// to avoid the need for buckets of size > 1.

import (
	"bytes"
	"fmt"
	//"math/big"
	//"os"
	"go/token"
	"sort"

	"golang.org/x/tools/go/ast/astutil"
)

// Idom returns the block that immediately dominates b:
// its parent in the dominator tree, if any.
// Neither the entry node (b.Index==0) nor recover node
// (b==b.Parent().Recover()) have a parent.
//

func contains(s []*Block, e *Block) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

//Consider all inverted CFG edges (x,c) such that x does not
//dominate c (therefore, c is a branch node)
//• Traverse the post‐dominator tree bottom‐up –n=x
//– while (n != parent of c in the post‐dominator tree) • report that n is control dependent on c
//• n = parent of n in the post‐dominator tree

func (invC *CFG) FindControlDeps() {

	// CFG is inverted
	for _, x := range invC.BlockSlice {
		for _, c := range x.SuccBs {
			if !x.Dominates(c) {
				n := x
				for n != c.dom.idom && n != nil {
					n.ControlDep = append(n.ControlDep, c)
					c.ControlDepee = append(c.ControlDepee, n)
					n = n.dom.idom
				}
			}
		}
	}

}

func (c *CFG) BuildPostDomTree() *CFG {
	c.InitializeBlocks()
	invC := c.InvertCFG()
	BuildDomTree(invC)
	return invC
}

func (b *Block) String() string { return astutil.NodeDescription(b.Stmt) }

func (b *Block) Idom() *Block { return b.dom.idom }

// Dominees returns the list of blocks that b immediately dominates:
// its children in the dominator tree.
//
func (b *Block) Dominees() []*Block { return b.dom.children }

// Dominates reports whether b dominates c.
func (b *Block) Dominates(c *Block) bool {
	return b.dom.pre <= c.dom.pre && c.dom.post <= b.dom.post
}

type byDomPreorder []*Block

func (a byDomPreorder) Len() int           { return len(a) }
func (a byDomPreorder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDomPreorder) Less(i, j int) bool { return a[i].dom.pre < a[j].dom.pre }

// DomPreorder returns a new slice containing the blocks of f in
// dominator tree preorder.
//
func (c *CFG) DomPreorder() []*Block {
	n := len(c.BlockSlice)
	order := make(byDomPreorder, n, n)
	copy(order, c.BlockSlice)
	sort.Sort(order)
	return order
}

// domInfo contains a Block's dominance information.
type domInfo struct {
	idom      *Block   // immediate dominator (parent in domtree)
	children  []*Block // nodes immediately dominated by this one
	pre, post int32    // pre- and post-order numbering within domtree
}

// ltState holds the working state for Lengauer-Tarjan algorithm
// (during which domInfo.pre is repurposed for CFG DFS preorder number).
type ltState struct {
	// Each slice is indexed by b.Index.
	sdom     []*Block // b's semidominator
	parent   []*Block // b's parent in DFS traversal of CFG
	ancestor []*Block // b's ancestor with least sdom
}

// dfs implements the depth-first search part of the LT algorithm.
func (lt *ltState) dfs(v *Block, i int32, preorder []*Block) int32 {
	preorder[i] = v
	v.dom.pre = i // For now: DFS preorder of spanning tree of CFG
	i++
	lt.sdom[v.Index] = v
	lt.link(nil, v)
	for _, w := range v.SuccBs {
		if lt.sdom[w.Index] == nil {
			lt.parent[w.Index] = v
			i = lt.dfs(w, i, preorder)
		}
	}
	return i
}

// eval implements the EVAL part of the LT algorithm.
func (lt *ltState) eval(v *Block) *Block {
	// TODO(adonovan): opt: do path compression per simple LT.
	u := v
	for ; lt.ancestor[v.Index] != nil; v = lt.ancestor[v.Index] {
		if lt.sdom[v.Index].dom.pre < lt.sdom[u.Index].dom.pre {
			u = v
		}
	}
	return u
}

// link implements the LINK part of the LT algorithm.
func (lt *ltState) link(v, w *Block) {
	lt.ancestor[w.Index] = v
}

// buildDomTree computes the dominator tree of f using the LT algorithm.
// Precondition: all blocks are reachable (e.g. optimizeBlocks has been run).
//
func BuildDomTree(c *CFG) {
	// The step numbers refer to the original LT paper; the
	// reordering is due to Georgiadis.

	// Clear any previous domInfo.
	for _, b := range c.BlockSlice {
		b.dom = domInfo{}
	}

	n := len(c.BlockSlice)
	// Allocate space for 5 contiguous [n]*Block arrays:
	// sdom, parent, ancestor, preorder, buckets.
	space := make([]*Block, 5*n, 5*n)
	lt := ltState{
		sdom:     space[0:n],
		parent:   space[n : 2*n],
		ancestor: space[2*n : 3*n],
	}

	// Step 1.  Number vertices by depth-first preorder.
	preorder := space[3*n : 4*n]
	root := c.BlockSlice[0]
	_ = lt.dfs(root, 0, preorder)

	buckets := space[4*n : 5*n]
	copy(buckets, preorder)

	// In reverse preorder...
	for i := int32(n) - 1; i > 0; i-- {
		w := preorder[i]
		//fmt.Println(preorder)
		// Step 3. Implicitly define the immediate dominator of each node.
		for v := buckets[i]; v != w; v = buckets[v.dom.pre] {
			u := lt.eval(v)
			if lt.sdom[u.Index].dom.pre < i {
				v.dom.idom = u
			} else {
				v.dom.idom = w
			}
		}
		// Step 2. Compute the semidominators of all nodes.
		lt.sdom[w.Index] = lt.parent[w.Index]
		for _, v := range w.PredBs {
			u := lt.eval(v)
			if lt.sdom[u.Index].dom.pre < lt.sdom[w.Index].dom.pre {
				lt.sdom[w.Index] = lt.sdom[u.Index]
			}
		}

		lt.link(lt.parent[w.Index], w)

		if lt.parent[w.Index] == lt.sdom[w.Index] {
			w.dom.idom = lt.parent[w.Index]
		} else {
			buckets[i] = buckets[lt.sdom[w.Index].dom.pre]
			buckets[lt.sdom[w.Index].dom.pre] = w
		}
	}

	// The final 'Step 3' is now outside the loop.
	for v := buckets[0]; v != root; v = buckets[v.dom.pre] {
		v.dom.idom = root
	}

	// Step 4. Explicitly define the immediate dominator of each
	// node, in preorder.
	for _, w := range preorder[1:] {
		if w == root {
			w.dom.idom = nil
		} else {
			if w.dom.idom != lt.sdom[w.Index] {
				w.dom.idom = w.dom.idom.dom.idom
			}
			// Calculate Children relation as inverse of Idom.
			w.dom.idom.dom.children = append(w.dom.idom.dom.children, w)
		}
	}

	_, _ = numberDomTree(root, 0, 0)

	// printDomTreeDot(os.Stderr, f)        // debugging
	// printDomTreeText(os.Stderr, root, 0) // debugging
}

// numberDomTree sets the pre- and post-order numbers of a depth-first
// traversal of the dominator tree rooted at v.  These are used to
// answer dominance queries in constant time.
//
func numberDomTree(v *Block, pre, post int32) (int32, int32) {
	v.dom.pre = pre
	pre++
	for _, child := range v.dom.children {
		pre, post = numberDomTree(child, pre, post)
	}
	v.dom.post = post
	post++
	return pre, post
}

// Testing utilities ----------------------------------------

// sanityCheckDomTree checks the correctness of the dominator tree
// computed by the LT algorithm by comparing against the dominance
// relation computed by a naive Kildall-style forward dataflow
// analysis (Algorithm 10.16 from the "Dragon" book).
//
//func sanityCheckDomTree(f *Function) {
//	n := len(f.Blocks)

//	// D[i] is the set of blocks that dominate f.Blocks[i],
//	// represented as a bit-set of block indices.
//	D := make([]big.Int, n)

//	one := big.NewInt(1)

//	// all is the set of all blocks; constant.
//	var all big.Int
//	all.Set(one).Lsh(&all, uint(n)).Sub(&all, one)

//	// Initialization.
//	for i, b := range f.Blocks {
//		if i == 0 || b == f.Recover {
//			// A root is dominated only by itself.
//			D[i].SetBit(&D[0], 0, 1)
//		} else {
//			// All other blocks are (initially) dominated
//			// by every block.
//			D[i].Set(&all)
//		}
//	}

//	// Iteration until fixed point.
//	for changed := true; changed; {
//		changed = false
//		for i, b := range f.Blocks {
//			if i == 0 || b == f.Recover {
//				continue
//			}
//			// Compute intersection across predecessors.
//			var x big.Int
//			x.Set(&all)
//			for _, pred := range b.PredBs {
//				x.And(&x, &D[pred.Index])
//			}
//			x.SetBit(&x, i, 1) // a block always dominates itself.
//			if D[i].Cmp(&x) != 0 {
//				D[i].Set(&x)
//				changed = true
//			}
//		}
//	}

//	// Check the entire relation.  O(n^2).
//	// The Recover block (if any) must be treated specially so we skip it.
//	ok := true
//	for i := 0; i < n; i++ {
//		for j := 0; j < n; j++ {
//			b, c := f.Blocks[i], f.Blocks[j]
//			if c == f.Recover {
//				continue
//			}
//			actual := b.Dominates(c)
//			expected := D[j].Bit(i) == 1
//			if actual != expected {
//				fmt.Fprintf(os.Stderr, "dominates(%s, %s)==%t, want %t\n", b, c, actual, expected)
//				ok = false
//			}
//		}
//	}

//	preorder := f.DomPreorder()
//	for _, b := range f.Blocks {
//		if got := preorder[b.dom.pre]; got != b {
//			fmt.Fprintf(os.Stderr, "preorder[%d]==%s, want %s\n", b.dom.pre, got, b)
//			ok = false
//		}
//	}

//	if !ok {
//		panic("sanityCheckDomTree failed for " + f.String())
//	}

//}

// Printing functions ----------------------------------------

// printDomTree prints the dominator tree as text, using indentation.
func printDomTreeText(buf *bytes.Buffer, v *Block, indent int) {
	fmt.Fprintf(buf, "%*s%s\n", 4*indent, "", v)
	for _, child := range v.dom.children {
		printDomTreeText(buf, child, indent+1)
	}
}

// printDomTreeDot prints the dominator tree of f in AT&T GraphViz
// (.dot) format.
func PrintDomTreeDot(buf *bytes.Buffer, c *CFG, fset *token.FileSet) {
	fmt.Fprintln(buf, "digraph domtree {")
	for i, b := range c.BlockSlice {
		v := b.dom
		fmt.Fprintf(buf, "\tn%d [label=\"%d (%d, %d)\",shape=\"rectangle\"];\n", v.pre, fset.Position(b.Stmt.Pos()).Line, v.pre, v.post)
		// TODO(adonovan): improve appearance of edges
		// belonging to both dominator tree and CFG.

		// Dominator tree edge.
		if i != 0 {
			fmt.Fprintf(buf, "\tn%d -> n%d [style=\"solid\",weight=100];\n", v.idom.dom.pre, v.pre)
		}
		// CFG edges.
		for _, pred := range b.PredBs {
			fmt.Fprintf(buf, "\tn%d -> n%d [style=\"dotted\",weight=0];\n", pred.dom.pre, v.pre)
		}
	}
	fmt.Fprintln(buf, "}")
}
