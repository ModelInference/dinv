package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

var (
	command   string
	directory string
	logger    *log.Logger
	filenames map[string][]string
)

func main() {
	Init()
	bucketFiles()
	//test Daikon
	i := 1
	for _, traces := range filenames {
		arg := append([]string{"daikon.Daikon"}, traces...)
		var extraArgs = []string{"--output_num_samples"}
		cmd := exec.Command("java", append(arg, extraArgs...)...)
		////cmd := exec.Command("java", "daikon.Daikon", traces[0])
		var buf bytes.Buffer
		cmd.Stdout = &buf
		err := cmd.Run()
		if err != nil {
			logger.Println(err)
		}
		//don't print no empty invariants
		if len(buf.String()) > 128 {
			logger.Println(buf.String())
		}
		logger.Printf("[%d / %d] Daikon Files Complete", i, len(filenames))
		i++
	}

}

func Init() {
	logger = log.New(os.Stdout, "["+os.Args[0]+"] ", log.Lshortfile)
	parseArgs()
	filenames = make(map[string][]string)
}

func parseArgs() {
	//default command top
	flag.StringVar(&command, "command", "top", "command to run on daikon logs [top,latest,volated]")
	//default the current working directory
	defaultDir, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}
	flag.StringVar(&directory, "directory", defaultDir, "directory containing run archive (typically log_archive)")
	flag.Parse()
}

func bucketFiles() {
	//ls all of the directories
	dirs, err := ioutil.ReadDir(directory)
	if err != nil {
		logger.Fatal(err)
	}
	//check all dirs for daikon output
	for _, dir := range dirs {
		if dir.IsDir() {
			//read each sub dir
			entries, err := ioutil.ReadDir(dir.Name())
			if err != nil {
				logger.Fatal(err)
			}
			for _, entry := range entries {
				full := directory + "/" + dir.Name() + "/" + entry.Name()
				ext := path.Ext(full)
				if ext == ".dtrace" {
					noextension := strings.Trim(entry.Name(), ext)
					//TODO do something smart with the file size or
					//something else to trim the output.
					//For now just add all files that have a dtrace,
					//assume that an inv.gz also exists but dont worry
					//for now.
					_, ok := filenames[noextension]
					if !ok {
						filenames[noextension] = make([]string, 0)
					}
					filenames[noextension] = append(filenames[noextension], full)
				}
			}

		}
	}
	/*
		for file := range filenames {
			for index := range filenames[file] {
				logger.Println(filenames[file][index])
			}
		}
	*/
}
