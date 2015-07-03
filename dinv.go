// Dinv

package main

import (
	"flag"
	"fmt"
	"os"

	"bitbucket.org/bestchai/dinv/instrumenter"
	"bitbucket.org/bestchai/dinv/logmerger"
)

var (
	inst    bool
	lm      bool
	verbose = true
)

func main() {
	flag.BoolVar(&inst, "instrumenter", false, "go run dinv -instrumenter file1 > instrumented.go")
	flag.BoolVar(&lm, "logmerger", false, "go run dinv -logmerger file1 file2 ...")
	flag.Parse()
	files := flag.Args()

	if lm {
		for i := 0; i < len(files); i++ {
			exists, err := fileExists(files[i])
			if !exists {
				err := fmt.Errorf("the file %s, does not exist\n%s\n", files[i], err)
				panic(err)
			}
		}
		if verbose {
			fmt.Printf("Merging Files...")
		}
		logmerger.Merge(files)
		if verbose {
			fmt.Printf("Complete\n")
		}
	}

	if inst {
		valid, err := validinstrumentationFiles(files)
		if !valid {
			panic(err)
		}
		if verbose {
			//fmt.Printf("Insturmenting %s...", files[0])
		}
		instrumenter.Instrument(files[0])
		if verbose {
			//fmt.Printf("Complete\n")
		}
	}
}

func validinstrumentationFiles(files []string) (bool, error) {
	if len(files) != 1 {
		return false, fmt.Errorf("please supply a single file for instrumenting")
	}
	exists, err := fileExists(files[0])
	if !exists {
		return false, fmt.Errorf("the file %s, does not exist\n%s\n", os.Args[1], err)
	}
	return true, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
