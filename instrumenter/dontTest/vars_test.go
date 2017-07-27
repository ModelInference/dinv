package instrumenter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/loader"
)

var astCommentFile *ast.File
var cfgs []*CFGWrapper

var clientDump [][]string = [][]string{
	[]string{"RUNS", "cTerm1", "sTerm1", "printErr", "Init", "ADDITION_ARGS", "sConn", "main", "cConn", "sSum", "done", "MarshallInts", "Logger", "cSum", "LARGEST_TERM", "buf", "cTerm2", "sTerm2", "Client", "Server", "UnmarshallInts", "SIZEOFINT"},
}

func TestVars(t *testing.T) {
	setup(t)

	//get dump nodes
	dumpNodes := GetDumpNodes(astCommentFile)

	//nonOptimized Collection
	var generated_code []string
	for i, dump := range dumpNodes {
		line := cfgs[0].fset.Position(dump.Pos()).Line
		// log all vars
		vars := GetAccessibleVarsInScope(int(dump.Pos()), astCommentFile, cfgs[0].fset)
		for j, _ := range vars {
			if !contains(vars[j], clientDump[i]) {
				t.Errorf("inconsistent Variables found <%s>\n{%s} =/= wanted {%s}\n", vars[j], vars, clientDump[i])
				t.Fatal()
			}
		}
		for k, _ := range clientDump[i] {
			if !contains(clientDump[i][k], vars) {
				t.Errorf("wanted variable not found <%s>\n{%s} =/= wanted {%s}\n", clientDump[i][k], vars, clientDump[i])
				t.Fatal()
			}
		}
		code := GenerateDumpCode(vars, line)
		generated_code = append(generated_code, code)
	}
	//fmt.Printf("%s\n", generated_code)

}

func writeFile(source string, filename string) string {
	pwd, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	_, name := filepath.Split(filename)
	modFilename := fmt.Sprintf("%s%s", pwd, name)
	file, _ := os.Create(modFilename)
	fmt.Printf("Writing file %s\n", modFilename)
	file.WriteString(source)
	file.Close()
	return modFilename
}

const source = `package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"

	"github.com/arcaneiceman/GoVector/govec"
)

const (
	SIZEOFINT     = 4
	ADDITION_ARGS = 2
	LARGEST_TERM  = 100
	RUNS          = 3
)

var (
	cConn *net.UDPConn
	sConn net.PacketConn

	buf                  [1024]byte
	cTerm1, cTerm2, cSum int
	sTerm1, sTerm2, sSum int
	done                 chan int
)

func main() {
	Init()
	go Server()
	go Client()
	<-done
	os.Exit(0)
}

func Client() {
	for t := 0; t < RUNS; t++ {
		cTerm1, cTerm2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := MarshallInts([]int{cTerm1, cTerm2})
		if cTerm1 < 5 {	//dummy should not be picked up
			dummy := 6
			print(dummy)
		}
		// sending UDP packet to specified address and port
		_, errWrite := cConn.Write(Logger.PrepareSend("", msg))
		for i := 0; i< cTerm1 ;i++{
			print(i)
		}
		//@dump
		printErr(errWrite)
		//adding local events for testing lattice /jan 23 2015
		//		for i := 0; i < 3; i++ {
		//			Logger.LogLocalEvent("Twittle Thumbs")
		//		}
		// Reading the response message

		_, errRead := cConn.Read(buf[0:])
		ret := Logger.UnpackReceive("Received", buf[0:])
		printErr(errRead)

		uret := UnmarshallInts(ret)
		cSum = uret[0]
		fmt.Printf("C: %d + %d = %d\n", cTerm1, cTerm2, cSum)
		cSum = 0
	}
	done <- 0
}

func Server() {
	for t := 0; t < RUNS; t++ {
		var buf [1024]byte
		var sTerm1, sTerm2, sSum int

		_, addr, err := sConn.ReadFrom(buf[0:])
		args := Logger.UnpackReceive("Received", buf[0:])
		printErr(err)
		uArgs := UnmarshallInts(args)
		sTerm1, sTerm2 = uArgs[0], uArgs[1]
		sSum = sTerm1 + sTerm2
		fmt.Printf("S: %d + %d = %d\n", sTerm1, sTerm2, sSum)
		msg := MarshallInts([]int{sSum})
		sConn.WriteTo(Logger.PrepareSend("Sending", msg), addr)
	}
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func MarshallInts(args []int) []byte {
	var i, j uint
	marshalled := make([]byte, len(args)*SIZEOFINT, len(args)*SIZEOFINT)
	for j = 0; int(j) < len(args); j++ {
		for i = 0; i < SIZEOFINT; i++ {
			marshalled[(j*SIZEOFINT)+i] = byte(args[j] >> ((SIZEOFINT - 1 - i) * 8))
		}
	}
	l := int(i)
	k := int(j)
	l = l + k
	print(l)
	return marshalled
}

func UnmarshallInts(args []byte) []int {
	var i, j uint
	unmarshalled := make([]int, len(args)/SIZEOFINT, len(args)/SIZEOFINT)
	for j = 0; int(j) < len(args)/SIZEOFINT; j++ {
		for i = 0; i < SIZEOFINT; i++ {
			unmarshalled[j] += int(args[SIZEOFINT*(j+1)-1-i] << (i * 8))
		}
	}
	l := int(i)
	k := int(j)
	l = l + k
	print(l)
	return unmarshalled
}

func Init() {
	Logger = govec.InitGoVector("self", "self.log")
	//setup receiving connection
	sConn, _ = net.ListenPacket("udp", ":8080")

	//Set up sending connection Address
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)
	cConn, _ = net.DialUDP("udp", lAddr, rAddr)

	done = make(chan int)
}

var Logger *govec.GoLog`

func setup(t *testing.T) {
	//load source
	var config loader.Config
	fset := token.NewFileSet()

	//write source out to file
	filename := writeFile(source, "source.go")
	f, err := config.ParseFile(filename, nil)
	if err != nil {
		t.Errorf("Encountered Error %s", err)
	}
	//silly global astfile bull
	astCommentFile, _ = parser.ParseFile(fset, "", source, parser.ParseComments)

	config.CreateFromFiles("testing", f)
	prog, err := config.Load()
	if err != nil {
		t.Error("Cannot Load")
	}

	//create a cfg for every function
	cfgs = make([]*CFGWrapper, 0)
	for i := 0; i < len(f.Decls); i++ {
		functionDec, ok := f.Decls[i].(*ast.FuncDecl)
		if ok {
			print("FuncFound\n")
			wrap := getWrapper(t, f, functionDec, prog)
			cfgs = append(cfgs, wrap)
		}
	}
}

func contains(s string, arr []string) bool {
	for i := range arr {
		if arr[i] == s {
			return true
		}
	}
	return false
}
