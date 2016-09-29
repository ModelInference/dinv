// Dinv

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime/pprof"

	"bitbucket.org/bestchai/dinv/instrumenter"
	"bitbucket.org/bestchai/dinv/logmerger"
)

const (
	//instrumenter defaults
	defaultFilename  = ""
	defaultDirectory = ""

	//logmerger defaults
	defaultMergePlan  = "TOLN" //total order line number merge
	defaultSampleRate = 100    //take 100 percent of all mined cuts

	defaultRenameingScheme = "" //rename hosts as fruit

)

var (
	inst   bool
	logmer bool

	//options for instrumenting
	clean            bool
	dataflowAnalysis bool
	dumpsLocalEvents bool
	directory        string
	file             string

	//log merger options
	shivizLog          bool
	mergePlan          string
	sampleRate         int
	totallyOrderedCuts bool
	renamingScheme     string

	//options for both
	verbose    bool
	debug      bool
	cpuprofile string

	logger *log.Logger
)

func setFlags() {
	flag.BoolVar(&inst, "instrumenter", false, "go run dinv -instrumenter -dir=directory")
	flag.BoolVar(&inst, "i", false, "go run dinv -i -dir=directory")

	flag.BoolVar(&clean, "c", false, "go run dinv -i -c removes converts insturmented dump code into commments")
	flag.BoolVar(&dataflowAnalysis, "df", false, "-df=true triggers dataflow analysis at dumpstatements")
	flag.BoolVar(&dumpsLocalEvents, "local", false, "-local=true logs //@dump annotations as local events")
	flag.StringVar(&directory, "dir", defaultDirectory, "-dir=directoryName recursivly instruments a directory inplace, original directory is duplicated for safty")
	flag.StringVar(&file, "file", defaultFilename, "-file=filename insturments a file")

	flag.BoolVar(&shivizLog, "shiviz", false, "-shiviz adds shiviz log to output")
	flag.StringVar(&mergePlan, "plan", defaultMergePlan, "-plan=TOLN merges based on total order, and line number\nOptions\n"+planDiscription)
	flag.IntVar(&sampleRate, "sample", defaultSampleRate, "-sample=50 % sample of consistant cuts to be analysed")
	flag.BoolVar(&totallyOrderedCuts, "toc", false, "-toc overlapping cuts are not analysed")
	flag.StringVar(&renamingScheme, "name", defaultRenameingScheme, "-name=color names hosts after colors includes color/fruit/philosopher")

	flag.BoolVar(&logmer, "logmerger", false, "go run dinv -logmerger file1 file2 ...")
	flag.BoolVar(&logmer, "l", false, "go run dinv -l file1 file2 ...")

	flag.BoolVar(&verbose, "verbose", false, "-verbose logs extensive output")
	flag.BoolVar(&verbose, "v", false, "-v logs extensive output")
	flag.BoolVar(&debug, "debug", false, "-debug adds pedantic level of logging")

	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.Parse()
}

func main() {
	setFlags()

	options := make(map[string]string)
	//set options relevent to all programs
	if verbose {
		logger = log.New(os.Stdout, "logger: ", log.Lshortfile)
	} else {
		var buf bytes.Buffer
		logger = log.New(&buf, "logger: ", log.Lshortfile)
	}

	if debug {
		options["debug"] = "on"
	}

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Insturmenter option setting
	if inst {
		//TODO complain about arguments not ment for the instrumenter

		//filechecking //exclusive or with filename and directory
		if file == defaultFilename && directory == defaultDirectory {
			logger.Fatalf("No directory or filename supplied for insturmentation, use either -file or -dir\n")
		} else if file != defaultFilename && directory != defaultDirectory {
			logger.Fatalf("Speficied filename =%s and directory = %s, use either -file or -dir\n", file, directory)
		}

		//test if file exists, if so add file option
		if file != defaultFilename {
			exists, err := fileExists(file)
			if !exists {
				logger.Fatalf("Error: : %s\n", err.Error())
			}

			logger.Printf("Instrumenting File %s\n", file)
			options["file"] = file
		}

		// test if the directory is valid. If so add to options, else
		// error
		if directory != defaultDirectory {
			valid, err := validinstrumentationDir(directory)
			if !valid {
				logger.Fatalf("Invalid Directory Error: %s\n", err.Error())
			}
			logger.Printf("Insturmenting Directory :%s\n", directory)
			options["directory"] = directory
		}

		if clean {
			options["clean"] = "on"
		}
		//set dataflow flag
		if dataflowAnalysis {
			options["dataflow"] = "on"
		}
		//set dump statements to log local events
		if dumpsLocalEvents {
			options["local"] = "on"
		}

		instrumenter.Instrument(options, logger)

		logger.Printf("Instrumentation Complete\n")
	}

	args := flag.Args()
	if logmer {
		//TODO complain about flags that should not be passed to the
		//log merger
		//check that all of the log files passed in exist
		for i := 0; i < len(args); i++ {
			exists, err := fileExists(args[i])
			if !exists {
				logger.Fatalf("the file %s, does not exist\n%s\n", args[i], err)
			}
		}
		//sort the logs into list of encoded point logs and govector
		//logs
		pointLogs, govecLogs, err := sortLogs(args)
		if err != nil {
			//report missing or corruped files
			logger.Fatalf("Log Sorting Error: %s", err.Error())
		}

		//set the merge plan
		options["mergePlan"] = mergePlan
		//set host id renaming scheme

		//set the sample rate and restrict to range (0 - 100)
		if sampleRate != defaultSampleRate {
			if sampleRate < 0 || sampleRate > 100 {
				logger.Printf("Waring sample rate %d out of range defaulting to %d", sampleRate, defaultSampleRate)
				sampleRate = defaultSampleRate
			}
		}
		options["sampleRate"] = fmt.Sprintf("%d", sampleRate)

		// flag to prevent the analysis of overlapping cuts
		if totallyOrderedCuts {
			options["totallyOrderedCuts"] = "on"
		}

		options["renamingScheme"] = renamingScheme

		if shivizLog {
			options["shiviz"] = "on"
		}

		logmerger.Merge(pointLogs, govecLogs, options, logger)
		logger.Printf("Complete\n")
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
	//NOTE if the file names change these regexes will need to .e
	//also the rely on the nanosecond timestamp ID as an identifier
	govecLogRegex := "(.+).log-Log.txt"
	pointLogRegex := "(.+)Encoded.txt"
	ErrorString := ""
	goReg := regexp.MustCompile(govecLogRegex)
	pointReg := regexp.MustCompile(pointLogRegex)
	for _, log := range logs {
		//check if the log is a govector log
		regexResults := goReg.FindStringSubmatch(log)
		if len(regexResults) == 2 && regexResults[1] != "" {
			id := regexResults[1]
			logger.Printf("reading in log: %d\n", id)
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
			//fmt.Println(id)
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
	isolatedGovecLogs := make([]string, 0)
	for pair := range logPairs {
		if logPairs[pair].point != "" && logPairs[pair].golog != "" {
			pointLogs = append(pointLogs, logPairs[pair].point)
			govecLogs = append(govecLogs, logPairs[pair].golog)
		}
		if logPairs[pair].golog == "" {
			ErrorString = ErrorString + logPairs[pair].point + ": Has no corresponding govecLog\n"
		}
		if logPairs[pair].point == "" {
			isolatedGovecLogs = append(isolatedGovecLogs, logPairs[pair].golog)
		}
		logger.Printf("Pair (%s,%s)\n", logPairs[pair].point, logPairs[pair].golog)
	}
	govecLogs = append(govecLogs, isolatedGovecLogs...)
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

var planDiscription = `TOLN: Totally ordered line number merge, logs are grouped by both the uniqueness of communication pattern, and the dump statements encountered
ETM: Entire cut merge, each cut is uniquely grouped by the total order of the communication within it (produces exponential files)
SRM: Send-Receive merge, hosts are paired by sends and receives which have matching dumps
SCM: Single Cut merge, every cut is grouped together, no totally ordering is considered (usefull for detecting invariants which are always present)
NONE: Vanilla Daikon merge, the hosts of the system are analysed indepenedently`
