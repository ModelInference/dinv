// Dinv

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"bitbucket.org/bestchai/dinv/instrumenter"
	"bitbucket.org/bestchai/dinv/logmerger"
)

var (
	inst    bool
	logmer  bool
	verbose bool

	logger *log.Logger
)

func setFlags() {
	flag.BoolVar(&inst, "instrumenter", false, "go run dinv -instrumenter directory packagename")
	flag.BoolVar(&inst, "i", false, "go run dinv -i directory packagename")
	flag.BoolVar(&logmer, "logmerger", false, "go run dinv -logmerger file1 file2 ...")
	flag.BoolVar(&logmer, "l", false, "go run dinv -l file1 file2 ...")
	flag.BoolVar(&verbose, "verbose", false, "-verbose logs extensive output")
	flag.BoolVar(&verbose, "v", false, "-verbose logs extensive output")
	flag.Parse()
}

func main() {
	setFlags()

	if verbose {
		logger = log.New(os.Stdout, "logger: ", log.Lshortfile)
	} else {
		var buf bytes.Buffer
		logger = log.New(&buf, "logger: ", log.Lshortfile)
	}

	args := flag.Args()
	if logmer {
		for i := 0; i < len(args); i++ {
			exists, err := fileExists(args[i])
			if !exists {
				err := fmt.Errorf("the file %s, does not exist\n%s\n", args[i], err)
				panic(err)
			}
		}
		if len(args)%2 != 0 {
			err := fmt.Errorf("please supply a govec log for each point log\n")
			panic(err)
		}
		logger.Printf("Merging Files...")
		pointLogs := make([]string, 0)
		govecLogs := make([]string, 0)
		for i := 0; i < len(args)/2; i++ {
			pointLogs = append(pointLogs, args[i])
			govecLogs = append(govecLogs, args[len(args)/2+i])
		}
		logmerger.Merge(pointLogs, govecLogs, logger)
		logger.Printf("Complete\n")
	}

	if inst {
		dir := args[0]
		packageName := args[1]
		valid, err := validinstrumentationDir(args[1:])
		if !valid {
			panic(err)
		}
		if verbose {
			fmt.Printf("Insturmenting %s...", args[0])
		}

		instrumenter.Instrument(dir, packageName, logger)
		if verbose {
			//fmt.Printf("Complete\n")
		}
	}
}

func validinstrumentationDir(args []string) (bool, error) {
	//TODO check that dir exists
	//TODO check for existing go args
	/*if len(args) != 3 {
		return false, fmt.Errorf("Directory or package non existant\n")
	}*/
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
