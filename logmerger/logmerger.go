/*
	logmerger is the main utility for merging the logs of programs
	involved in distributed computation. The input to the log merger
	is a set of filenames corresponding to the logs of distributed
	programs. The output of the log merger is a set of files that
	Daikon can use for invarient detection in the logs. The log merger
	can merge the program points of the logs in various ways based on
	a user defined specification

	Author: Stewart Grant
	Edited: July 6 2015
*/
package logmerger

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/wantonsolutions/GoVector/govec/vclock"
)

var (
	logger *log.Logger
)

func Init() {
	if logger == nil {
		logger = log.New(os.Stdout, "logger: ", log.Lshortfile)
	}
}

//Merge is the control fuction for log merging. The input is an array
//of strings corresponding to the log files which will be merged. The
//output is a set of Daikon files
func Merge(logfiles []string, gologfiles []string, inlogger *log.Logger) {
	Init()
	logger = inlogger
	for _, log := range logfiles {
		logger.Println(log)
	}
	for _, log := range gologfiles {
		logger.Println(log)
	}
	logs := buildLogs(logfiles, gologfiles)
	states := mineStates(logs)
	spec := &MergeSpec{totalOrderLineNumberMerge, 100, false}
	writeTraceFiles(states, spec)
}

//buildLogs parses the log files into a 2D array of program points,
//one array per file. The logs are preprocesses, by appending their
//id's to their variable names, and injecting a zeroed vector clock at
//the begining of each log, to act as a base case during computation
func buildLogs(logFiles []string, gologFiles []string) [][]Point {
	logs := make([][]Point, 0)
	goLogs := make([]*golog, 0)
	for i := 0; i < len(logFiles); i++ {
		log := readLog(logFiles[i])
		logs = append(logs, log)
	}
	for i := 0; i < len(gologFiles); i++ {
		goLog, err := ParseGologFile(gologFiles[i])
		if err != nil {
			panic(err)
		}
		goLogs = append(goLogs, goLog)
	}
	clocks, _ := VectorClockArraysFromLogs(logs)
	logger.Printf("Found %d seperate clocks\n", len(clocks))
	ids := idClockMapper(clocks)
	for i, log := range logs {
		addNodeName(ids[i], log)
		for _, goLog := range goLogs {
			if goLog.id == ids[i] { //match the ids of the logs
				log = injectMissingPoints(log, goLog)
			}
		}
	}
	return logs
}

func injectMissingPoints(points []Point, log *golog) []Point {
	pointIndex, gologIndex := 0, 0
	injectedPoints := make([]Point, 0)
	for pointIndex < len(points) {
		logger.Printf("point index :%d maxp :%d maxc: %d\n", pointIndex, len(points), len(log.clocks))
		pointClock, _ := vclock.FromBytes(points[pointIndex].VectorClock)
		ticks, _ := pointClock.FindTicks(log.id)
		if int(ticks) == gologIndex {
			injectedPoints = append(injectedPoints, points[pointIndex])
			pointIndex += 2
		} else {
			newPoint := new(Point)
			newPoint.VectorClock = log.clocks[gologIndex].Bytes()
			injectedPoints = append(injectedPoints, *newPoint)
		}
		gologIndex++
	}
	//fill in all unlogged clocks
	for gologIndex < len(log.clocks) {
		newPoint := new(Point)
		newPoint.VectorClock = log.clocks[gologIndex].Bytes()
		injectedPoints = append(injectedPoints, *newPoint)
		gologIndex++
	}
	return injectedPoints
}

//addBaseLog Injects a single valued vector clock as the base entry of
//a log. The base clock acts as a uniform starting point for
//computations being done to the logs.
func addBaseLog(name string, log []Point) []Point {
	clock := vclock.New()
	clock.Update(name, 0)
	first := new(Point)
	first.VectorClock = clock.Bytes()
	baseLog := make([]Point, 0)
	baseLog = append(baseLog, *first)
	for i := range log {
		baseLog = append(baseLog, log[i])
	}
	return baseLog

}

//addNodeName appends the name of the log file to the beginning of
//each variable in the log.
//TODO extend this naming scheme to classifiy variable names on a cut
//and interaction basis.
func addNodeName(name string, logs []Point) {
	for i := range logs {
		for j := range logs[i].Dump {
			logs[i].Dump[j].VarName = name + "-" + logs[i].Dump[j].VarName
		}
	}
}

//writeLogToFile produces a daikon dtrace file based on a log
//represented as an array of points
func writeLogToFile(log []Point, filename string) {
	filenameWithExtenstion := fmt.Sprintf("%s.dtrace", filename)
	file, _ := os.Create(filenameWithExtenstion)
	mapOfPoints := createMapOfLogsForEachPoint(log)
	writeDeclaration(file, mapOfPoints)
	writeValues(file, log)
}

//createMapOfLogsForEachPoint buckets points based on the line number
//they occur on. The map corresponding to each unique line number is
//returned
func createMapOfLogsForEachPoint(log []Point) map[string][]Point {
	mapOfPoints := make(map[string][]Point, 0)
	for i := 0; i < len(log); i++ {
		mapOfPoints[log[i].Id] = append(mapOfPoints[log[i].Id], log[i])
	}
	return mapOfPoints
}

//writeDeclaration writes out variable names and their types to the
//specified open file. The declarations are in a Daikon readable
//format
func writeDeclaration(file *os.File, mapOfPoints map[string][]Point) {
	file.WriteString("decl-version 2.0\n")
	file.WriteString("var-comparability none\n")
	file.WriteString("\n")
	for _, v := range mapOfPoints {
		point := v[0]
		file.WriteString(fmt.Sprintf("ppt p-%s:::%s\n", point.Id, point.Id))
		file.WriteString(fmt.Sprintf("ppt-type point\n"))
		for i := 0; i < len(point.Dump); i++ {
			file.WriteString(fmt.Sprintf("variable %s\n", point.Dump[i].VarName))
			file.WriteString(fmt.Sprintf("var-kind variable\n"))
			file.WriteString(fmt.Sprintf("dec-type %s\n", point.Dump[i].Type))
			file.WriteString(fmt.Sprintf("rep-type %s\n", point.Dump[i].Type))
			file.WriteString(fmt.Sprintf("comparability -1\n"))
		}
		file.WriteString("\n")

	}
}

//writeValeus outputs variable values and their associated line
//numbers. The output is in a Daikon readable format.
func writeValues(file *os.File, log []Point) {
	for i := range log {
		point := log[i]
		file.WriteString(fmt.Sprintf("p-%s:::%s\n", point.Id, point.Id))
		file.WriteString(fmt.Sprintf("this_invocation_nonce\n"))
		file.WriteString(fmt.Sprintf("%d\n", i))
		for i := range point.Dump {
			variable := point.Dump[i]
			file.WriteString(fmt.Sprintf("%s\n", variable.VarName))
			if variable.Type == "int" {
				file.WriteString(fmt.Sprintf("%d\n", variable.Value))
			} else {
				file.WriteString(strings.Replace(fmt.Sprintf("%s", variable.Value), "\n", " ", -1) + "\n")
			}
			file.WriteString(fmt.Sprintf("1\n"))
		}
		file.WriteString("\n")

	}
}

//mineStates is a an assembly line method for producting all the structures
//nessisary for extracting distributed state. The input is an array of
//logs, and the ouput is a set of ordered states extraced from those
//logs.
func mineStates(logs [][]Point) []State {
	logger.Printf("\nStripping Clocks... ")
	clocks, _ := VectorClockArraysFromLogs(logs)
	logger.Printf("Done\nBuilding Lattice... ")
	lattice := BuildLattice(clocks)
	logger.Printf("Done\nCalculating Delta... ")
	deltaComm := enumerateCommunication(clocks)
	logger.Printf("Done\nMining Consistent Cuts... ")
	consistentCuts := mineConsistentCuts(lattice, clocks, deltaComm)
	logger.Printf("Done\nExtracting States... ")
	states := statesFromCuts(consistentCuts, clocks, logs)
	logger.Printf("Done\n")
	return states
}

//statesFromCuts constructs an array of ordered states, from a set of
//ordered cuts. The corresponding log of vector clocks is used to
//determing total ordering within a cut.
func statesFromCuts(cuts []Cut, clocks [][]vclock.VClock, logs [][]Point) []State {
	states := make([]State, 0)
	ids := idClockMapper(clocks)
	for _, cut := range cuts {
		state := &State{}
		state.Cut = cut
		for i, clock := range state.Cut.Clocks {
			found, index := searchClockById(clocks[i], &clock, ids[i])
			if found {
				state.Points = append(state.Points, logs[i][index])
			}
		}
		state.TotalOrdering = totalOrderFromCut(cut, clocks)
		logger.Printf("%s\n", state.String())
		states = append(states, *state)
	}
	return states
}

//writeTraceFiles constructs a set unique trace file based on several
//specifiations in the MergeSpec.
func writeTraceFiles(states []State, spec *MergeSpec) {
	if spec.totallyOrderedCuts {
		states = filterTotalOrder(states)
	}
	mergedPoints := spec.merger(states)
	written := make([][]bool, len(mergedPoints))
	for i := range mergedPoints {
		written[i] = make([]bool, len(mergedPoints[i]))
	}
	newFile := true
	for newFile {
		newFile = false
		var filename string
		pointLog := make([]Point, 0)
		for i := range mergedPoints {
			for j := range mergedPoints[i] {
				if !written[i][j] {
					if !newFile {
						logger.Printf("New file :%s\n", mergedPoints[i][j].Id)
						filename = mergedPoints[i][j].Id
						newFile = true
					}
					if filename == mergedPoints[i][j].Id {
						//sample rate
						if (rand.Int() % 100) < spec.sampleRate {
							pointLog = append(pointLog, mergedPoints[i][j])
						}
						written[i][j] = true
					}
				}
			}
		}
		if newFile {
			writeLogToFile(pointLog, filename)
		}
	}
}

//filterTotalOrder takes a set of states as an argumet, and fileters
//out conncurent states, such that the output states can be totaly
//ordered with respect to one another.
func filterTotalOrder(states []State) []State {
	filteredStates := make([]State, 0)
	filteredStates = append(filteredStates, states[0])
	for i := 1; i < len(states); i++ {
		if filteredStates[len(filteredStates)-1].Cut.HappenedBefore(states[i].Cut) {
			filteredStates = append(filteredStates, states[i])
			fmt.Printf("%s\n", states[i].String())
		}
	}
	return filteredStates
}

//totalOrderLineNumberMerge merges program points that participate in
//a total ordering within a cut. The output is a two dimentional array
//of merged program points [i][j] where the ith index corresponds to
//the i'th state in states, and the j'th index corresponds to the j'th
//total ordering on that state.
func totalOrderLineNumberMerge(states []State) [][]Point {
	mergedPoints := make([][]Point, len(states))
	for i, state := range states {
		mergedPoints[i] = make([]Point, len(state.TotalOrdering))
		for j := range state.TotalOrdering {
			points := make([]Point, 0)
			for k := range state.TotalOrdering[j] {
				points = append(points, state.Points[state.TotalOrdering[j][k]])
			}
			mergedPoints[i][j] = mergePoints(points)
			logger.Println("%s\n", mergedPoints[i][j].Id)
		}
	}
	return mergedPoints
}

//entireCutMerge merges all of the points in each individual cut. The
//return array of points [i][j] corresponds to the [i]th entry in
//states, and the jth cut. A side effect of this merge is j == 0
func entireCutMerge(states []State) [][]Point {
	mergedPoints := make([][]Point, len(states))
	for i, state := range states {
		mergedPoints[i] = make([]Point, 1)
		mergedPoints[i][0] = mergePoints(state.Points)
	}
	return mergedPoints
}

//sendReceiveMerge merges all sets of sends and receives in a cut. The
//return array of points [i][j] corresponds to the i'th entry in
//states, and the j'th send -> receive pair in that state.
func sendReceiveMerge(states []State) [][]Point {
	mergedPoints := make([][]Point, len(states))
	for i, state := range states {
		mergedPoints[i] = make([]Point, 0)
		for j := 0; j < len(state.TotalOrdering); j++ {
			for k := 0; k < len(state.TotalOrdering[j])-1; k++ {
				sendReceivePair := []Point{state.Points[state.TotalOrdering[j][k]], state.Points[state.TotalOrdering[j][k+1]]}
				mergedPoints[i] = append(mergedPoints[i], mergePoints(sendReceivePair))
			}
		}
	}
	return mergedPoints
}

//Merge Points merges an array of points into a single aggregated point
func mergePoints(points []Point) Point {
	var mergedPoint Point
	for _, point := range points {
		mergedPoint.Dump = append(mergedPoint.Dump, point.Dump...) //...
		logger.Printf("id:%s\n", point.Id)
		mergedPoint.Id = mergedPoint.Id + "_" + point.Id
		pVClock1, _ := vclock.FromBytes(mergedPoint.VectorClock)
		pVClock2, _ := vclock.FromBytes(point.VectorClock)
		temp := pVClock1.Copy()
		temp.Merge(pVClock2)
		mergedPoint.VectorClock = temp.Bytes()
	}
	return mergedPoint
}

//readLog attempts to extract an array of program points from a log
//file. If the log file does not exist or is unreadable, readLog
//panics. Otherwise an array of program points is returned
func readLog(filePath string) []Point {
	fileR, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	decoder := gob.NewDecoder(fileR)
	pointArray := make([]Point, 0)
	var e error = nil
	for e == nil {
		var decodedPoint Point
		e = decoder.Decode(&decodedPoint)
		if e == nil {
			//logger.Printf(decodedPoint.String())
			pointArray = append(pointArray, decodedPoint)
		}
	}
	return pointArray
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

//specifies how the program points should be merged. The merger
//function translates the set of states into a 2D array of program
//points. sampleRate is an integer 0-100 that specifies the % of
//states which should be analized. totallyOrdedCuts is a flag
//specifing if concurrent states should be analized
type MergeSpec struct {
	merger             func(state []State) [][]Point
	sampleRate         int
	totallyOrderedCuts bool
}

//Point is a representation of a program point. Name value pair is the
//variable values at that program point. LineNumber is the line the
//variables were gathered on. VectorClock is byte valued vector clock
//at that the time the program point was logged
type Point struct {
	Dump               []NameValuePair
	Id                 string
	VectorClock        []byte
	CommunicationDelta int
}

//Name value pair matches variable names to their values, along with
//their type
type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}

//String representation of a name value pair
func (nvp NameValuePair) String() string {
	return fmt.Sprintf("(%s,%s,%s)", nvp.VarName, nvp.Value, nvp.Type)
}

//String representation of a program point
func (p Point) String() string {
	return fmt.Sprintf("%s : %s", p.Id)
}

//fileExists returns true if the file specified by path exists. If not
//false is returned, along with an error.
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

type golog struct {
	id       string
	clocks   []*vclock.VClock
	messages []string
}

func (g *golog) String() string {
	var text string
	for i := range g.clocks {
		text += fmt.Sprintf("id: %s\t clock: %s\n %s\n", g.id, g.clocks[i].ReturnVCString(), g.messages[i])
	}
	return text
}

/* reading govec logs */
func ParseGologFile(filename string) (*golog, error) {
	var govecRegex string = "(\\S*) ({.*})\n(.*)"
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var text string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text += "\n" + scanner.Text()
	}
	logger.Print(text)
	log, err := LogsFromString(text, govecRegex)
	if err != nil {
		defer file.Close()
		return nil, err
	}
	logger.Print(log.String())

	return log, nil
}

func LogsFromString(clockLog, regex string) (*golog, error) {
	rex := regexp.MustCompile(regex)
	matches := rex.FindAllStringSubmatch(clockLog, -1)
	if len(matches) <= 0 {
		return nil, fmt.Errorf("No matches found")
	}
	id := matches[0][1]
	messages := make([]string, 0)
	rawClocks := make([]string, 0)
	for i := range matches {
		rawClocks = append(rawClocks, matches[i][2])
		messages = append(messages, matches[i][3])
	}

	vclocks := make([]*vclock.VClock, 0)
	for i := range rawClocks {
		clock, err := ClockFromString(rawClocks[i], "\"([A-Za-z0-9]+)\":([0-9]+)")
		if clock == nil || err != nil {
			return nil, err
		}
		vclocks = append(vclocks, clock)
	}
	log := &golog{id, vclocks, messages}
	return log, nil
}

func ClockFromString(clock, regex string) (*vclock.VClock, error) {
	re := regexp.MustCompile(regex)
	matches := re.FindAllStringSubmatch(clock, -1)
	ids := make([]string, 0)
	ticks := make([]int, 0)
	for i := range matches {
		ids = append(ids, matches[i][1])
		time, err := strconv.Atoi(matches[i][2])
		ticks = append(ticks, time)
		if err != nil {
			return nil, err
		}
	}
	extractedClock := ConstructVclock(ids, ticks)
	if extractedClock == nil {
		return nil, errors.New("unable to extract clock\n")
	}
	return extractedClock, nil
}
