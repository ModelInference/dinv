// Dinv

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

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
		pointLogs, govecLogs, err := sortLogs(args)
		if err != nil {
			panic(err)
		}
		logmerger.Merge(pointLogs, govecLogs, logger)
		logger.Printf("Complete\n")
	}

	if inst {
		dir := args[0]
		valid, err := validinstrumentationDir(args[0])
		if !valid {
			panic(err)
		}
		if verbose {
			fmt.Printf("Insturmenting %s...", args[0])
		}

		instrumenter.Instrument(dir, logger)
		if verbose {
			//fmt.Printf("Complete\n")
		}
	}
}

//sortLogs checks an array of filenames to ensure that they contain a
//set of logfiles. The list of logs should contain two unique logs each host,
//a point encoded log, and a govec log, identified by a shared id.
//Returned are two arrays of coresponding to the two log files, whose
//indexes correspond to the same host. An error is returned if A log
//is missing its pair, or if the name of the log was unpareable
func sortLogs(logs []string) ([]string, []string, error) {
	//structure for pairing poing logs and govec logs
	type logpair struct {
		point string
		golog string
	}
	logPairs := make(map[string]logpair)
	//regular expressions for parsing point and govec log filenames
	//NOTE if the file names change these regexes will need to change
	//also the rely on the nanosecond timestamp ID as an identifier
	govecLogRegex := "([0-9]+).log-Log.txt"
	pointLogRegex := ".+-([0-9]+)Encoded.txt"
	ErrorString := ""
	goReg := regexp.MustCompile(govecLogRegex)
	pointReg := regexp.MustCompile(pointLogRegex)
	for _, log := range logs {
		//check if the log is a govector log
		regexResults := goReg.FindStringSubmatch(log)
		if len(regexResults) == 2 && regexResults[1] != "" {
			id := regexResults[1]
			fmt.Println(id)
			_, ok := logPairs[id]
			//if a corresponding log exists pair the two
			if ok {
				logPairs[id] = logpair{logPairs[id].point, log}
			} else {
				logPairs[id] = logpair{"", log}
			}
			continue
		}
		//check if the log is an encoded point log
		regexResults = pointReg.FindStringSubmatch(log)
		if len(regexResults) == 2 && regexResults[1] != "" {
			id := regexResults[1]
			fmt.Println(id)
			_, ok := logPairs[id]
			if ok {
				logPairs[id] = logpair{log, logPairs[id].golog}
			} else {
				logPairs[id] = logpair{log, ""}
			}
			continue
		}
		//if the log is neither there is an error
		ErrorString = ErrorString + "Log:" + log + " could not be matched as a govec or point encoded log\n"
	}
	pointLogs := make([]string, 0)
	govecLogs := make([]string, 0)
	for pair := range logPairs {
		if logPairs[pair].point == "" {
			ErrorString = ErrorString + logPairs[pair].golog + ": Has no corresponding encoded point log\n"
		}
		if logPairs[pair].golog == "" {
			ErrorString = ErrorString + logPairs[pair].point + ": Has no corresponding govecLog\n"
		}
		fmt.Printf("Pair (%s,%s)\n", logPairs[pair].point, logPairs[pair].golog)
		pointLogs = append(pointLogs, logPairs[pair].point)
		govecLogs = append(govecLogs, logPairs[pair].golog)
	}
	if ErrorString == "" {
		return pointLogs, govecLogs, nil
	}
	return nil, nil, fmt.Errorf(ErrorString)
}

func validinstrumentationDir(dir string) (bool, error) {
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
