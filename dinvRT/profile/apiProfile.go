package main

import (
	"bitbucket.org/bestchai/dinv/dinvRT"
	"flag"
	"log"
	"os"
	"runtime/pprof"
)

var (
	i          int
	s          string
	b          bool
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

const RUNS = 100000

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for i := 0; i < RUNS; i++ {
		dinvRT.Dump("id", "int,string,bool", i, s, b)
	}
}
