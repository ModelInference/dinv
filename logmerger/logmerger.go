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
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"

	"github.com/arcaneiceman/GoVector/govec/vclock"
)

//specifies how the program points should be merged. The merger
//function translates the set of states into a 2D array of program
//points. sampleRate is an integer 0-100 that specifies the % of
//states which should be analized. totallyOrdedCuts is a flag
//specifing if concurrent states should be analized
var (
	logger *log.Logger
	debug  = false
	//produce a shiviz readable log
	shiviz = false
	//specifies how program points should be merged. The merging plan
	//translates the set of states into a 2D array of program points.
	mergePlan = func(states []State) [][]Point { return nil }
	//totally orderd cuts specifices that if a set of cuts are
	//concurrent with eachother, only one should be used for invariant
	//detection
	totallyOrderedCuts = false
	//sampleRate is an iteger 0-100 specifing the % of states which
	//should be analysed.
	sampleRate     = 0
	renamingScheme = ""

	hosts int
)

const (
	CLEAR_LINE = "\r																															"
)

func initalizeLogMerger(options map[string]string, inlogger *log.Logger) {
	if inlogger == nil {
		logger = log.New(os.Stdout, "instrumenter:", log.Lshortfile)
	} else {
		logger = inlogger
	}
	for setting := range options {
		switch setting {
		case "debug":
			debug = true
		case "mergePlan":
			switch options[setting] {
			case "TOLN":
				mergePlan = totalOrderLineNumberMerge
			case "ETM":
				mergePlan = entireCutMerge
			case "SRM":
				mergePlan = sendReceiveMerge
			case "SCM":
				logger.Println("using SCM plan")
				mergePlan = singleCutMerge
			case "NONE":
				mergePlan = noMerge
			default:
				mergePlan = func([]State) [][]Point { logger.Fatalf("Error Invalid Merge Plan"); return nil }
			}
		case "sampleRate":
			sampleRate, _ = strconv.Atoi(options[setting])
		case "totallyOrderedCuts":
			totallyOrderedCuts = true
		case "renamingScheme":
			renamingScheme = options[setting]
		case "shiviz":
			shiviz = true
		default:
			continue
		}
	}
}

//Merge is the control fuction for log merging. The input is an array
//of strings corresponding to the log files which will be merged. The
//output is a set of Daikon files
func Merge(logfiles []string, gologfiles []string, options map[string]string, inlogger *log.Logger) {
	initalizeLogMerger(options, inlogger)
	logs, goLogs := buildLogs(logfiles, gologfiles)
	plogs, pgoLogs := partitionLogs(logs, goLogs)
	//jump to writing
	if options["mergePlan"] == "NONE" {
		writeUnmergedTraces(logfiles, logs)
	} else {
		for i := range plogs {
			fmt.Printf("Merging group %d\n", i)
			states := mineStates(plogs[i], pgoLogs[i])
			writeTraceFiles(states)
		}
	}
}

//partition logs seperates logs that do not communicate into seperate
//arrays
func partitionLogs(points [][]Point, gos []*golog) ([][][]Point, [][]*golog) {
	seperatePoints := make([][][]Point, 0)
	seperategoLogs := make([][]*golog, 0)
	used := make([]bool, len(points))
	var checked int
	for checked < len(gos) {
		found := true
		ids := make(map[string]bool, 0)
		for found {
			found = false
			for i := 0; i < len(gos); i++ {
				if !used[i] {
					lastClock := gos[i].clocks[len(gos[i].clocks)-1]
					if len(ids) == 0 {
						for id := range lastClock {
							ids[id] = true
							found = true
							used[i] = true
							checked++
						}
					} else if ids[gos[i].id] {
						for id := range lastClock {
							ids[id] = true
							found = true
							used[i] = true
							checked++
						}
					}
				}
			}
			fmt.Println()
		}
		partLog := make([][]Point, 0)
		partGoLog := make([]*golog, 0)
		fmt.Printf("Group contains ")
		for i, log := range gos {
			if ids[log.id] {
				fmt.Printf("%s ", log.id)
				partLog = append(partLog, points[i])
				partGoLog = append(partGoLog, gos[i])
			}
		}
		fmt.Println()
		seperatePoints = append(seperatePoints, partLog)
		seperategoLogs = append(seperategoLogs, partGoLog)
	}
	return seperatePoints, seperategoLogs
}

//writeUnmergedTraces is used when daikon traces for individual hosts
//are wanted. The point logs passed in should not be merged
func writeUnmergedTraces(filenames []string, logs [][]Point) {
	for i, filename := range filenames {
		unmergedName := filename + "unmerged.log"
		logger.Printf("New unmerged trace %s\n", unmergedName)
		writeLogToFile(logs[i], unmergedName)
	}

}

//buildLogs parses the log files into a 2D array of program points,
//one array per file. The logs are preprocesses, by appending their
//id's to their variable names, and injecting a zeroed vector clock at
//the begining of each log, to act as a base case during computation
func buildLogs(logFiles []string, gologFiles []string) ([][]Point, []*golog) {
	logs := make([][]Point, 0)
	goLogs := make([]*golog, 0)
	for i := 0; i < len(logFiles); i++ {
		log := readLog(logFiles[i])
		goLog, err := ParseGologFile(gologFiles[i])
		if err != nil {
			panic(fmt.Sprintf("Corresponding goLogfile for logfile '%s' missing: %s", logFiles[i], err.Error))
		}
		logs = append(logs, log)
		goLogs = append(goLogs, goLog)
	}

	for i := range logs {
		//sort the log as a pre processesing step
		fmt.Printf("\rSorting Logs %d/%d", i+1, len(logs))
		sort.Sort(goLogs[i])
		logs[i] = injectMissingPoints(logs[i], goLogs[i])
	}

	//create pointLogs executions with only gologs
	for i := len(logFiles); i < len(goLogs); i++ {
		sort.Sort(goLogs[i])
		logs = append(logs, injectMissingPoints(make([]Point, 0), goLogs[i]))
	}
	fmt.Println()

	replaceIds(logs, goLogs, renamingScheme)

	if shiviz {
		writeShiVizLog(logs, goLogs)
	}

	return logs, goLogs
}

//mineStates is a an assembly line method for producting all the structures
//nessisary for extracting distributed state. The input is an array of
//logs, and the ouput is a set of ordered states extraced from those
//logs.
func mineStates(logs [][]Point, clockLogs []*golog) []State {
	logger.Printf("\nStripping Clocks... ")
	clocks, _ := VectorClockArraysFromGoVectorLogs(clockLogs)
	logger.Printf("Done\nBuilding Lattice... ")
	//NOTE REVERT TESTING
	lattice := BuildLattice5(clocks)
	//fmt.Println(ltest.Points)
	//lattice := BuildLattice4(clocks)
	//latticeB := BuildLattice(clocks)
	//CompareLattice(lattice,latticeB)
	//CompareLattice(latticeB,lattice)
	logger.Printf("Done\nCalculating Delta... ")
	deltaComm := enumerateCommunication(clocks)
	logger.Printf("Done\nMining Consistent Cuts... ")
	//consistentCuts := mineConsistentCuts(lattice, clocks, deltaComm)
	consistentCuts := mineConsistentCuts2(lattice, clocks, deltaComm)
	logger.Printf("Done\nExtracting States... ")
	states := statesFromCuts2(consistentCuts, clocks, logs)
	logger.Printf("Done\n")
	return states
}

//statesFromCuts constructs an array of ordered states, from a set of
//ordered cuts. The corresponding log of vector clocks is used to
//determing total ordering within a cut.
func statesFromCuts(cuts []Cut, clocks [][]vclock.VClock, logs [][]Point) []State {
	states := make([]State, 0)
	ids := idClockMapper(clocks)
	for _, id := range ids {
		logger.Println(id)
	}
	for cutIndex, cut := range cuts {
		state := &State{}
		state.Cut = cut
		for i, clock := range state.Cut.Clocks {
			found, index := searchClockById(clocks[i], clock, ids[i])
			//TODO deal with local events, empty and local events are
			if found {
				state.Points = append(state.Points, logs[i][index])
			} else {
				logger.Fatalf("UNABLE TO LOCATE LOG %s entry %d\n", ids[i], index)
				fmt.Printf("unfound log entry %s index %d\n", ids[i], index)
			}
		}
		state.TotalOrdering = totalOrderFromCut(cut, clocks)
		//logger.Printf("%s\n", state.String())
		fmt.Printf("\rExtracting states %3.0f%% \t[%d] found", 100*float32(cutIndex)/float32(len(cuts)), len(states))
		states = append(states, *state)
	}
	fmt.Println()
	return states
}

//statesFromCuts constructs an array of ordered states, from a set of
//ordered cuts. The corresponding log of vector clocks is used to
//determing total ordering within a cut.
func statesFromCuts2(cuts []Cut, clocks [][]vclock.VClock, logs [][]Point) []State {
	states := make([]State, 0)
	ids := idClockMapper(clocks)
	//mClocks := clocksToMaps(clocks) //clockWrapper
	mPoints := pointsToMaps(logs)
	for _, id := range ids {
		logger.Println(id)
	}
	for cutIndex, cut := range cuts {
		state := &State{}
		state.Cut = cut
		for i, clock := range state.Cut.Clocks {
			time, _ := clock.FindTicks(ids[i])
			point, found := mPoints[ids[i]][time]
			//TODO deal with local events, empty and local events are
			if found {
				state.Points = append(state.Points, point)
			} else {
				logger.Fatalf("UNABLE TO LOCATE LOG %s entry %s\n", ids[i], point.String())
				fmt.Printf("unfound log entry %s index %s\n", ids[i], point.String())
			}
		}
		state.TotalOrdering = totalOrderFromCut(cut, clocks) //SPEED UP
		logger.Printf("%s\n", state.String())
		fmt.Printf("\rExtracting states %3.0f%% \t[%d] found", 100*float32(cutIndex)/float32(len(cuts)), len(states))
		states = append(states, *state)
	}
	fmt.Println()
	return states
}

//writeTraceFiles constructs a set unique trace file based on several
//specifiations in the MergeSpec.
func writeTraceFiles(states []State) {
	totallyOrderedCuts = false
	// mergePlan = totalOrderLineNumberMerge
	sampleRate = 100

	logger.Printf("Writing Traces\n")
	if totallyOrderedCuts {
		states = filterTotalOrder(states)
	}
	fmt.Printf("length of states: %d", len(states))
	mergedPoints := mergePlan(states)
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
						if (rand.Int() % 100) < sampleRate {
							pointLog = append(pointLog, mergedPoints[i][j])
						}
						written[i][j] = true
					}
				}
			}
		}
		if newFile {
			logger.Printf("New trace file %s\n", filename)
			writeLogToFile(pointLog, filename)
		}
	}
}

//filterTotalOrder takes a set of states as an argumet, and fileters
//out conncurent states, such that the output states can be totaly
//ordered with respect to one another.
func filterTotalOrder(states []State) []State {
	logger.Println("Filtering states by total order\n")
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
	logger.Println("Merging points by line number and total order")
	mergedPoints := make([][]Point, len(states))

	for i, state := range states {
		fmt.Printf("\rMerging States %3.0f%%", float32(i)/float32(len(states))*100)
		mergedPoints[i] = make([]Point, len(state.TotalOrdering))
		for j := range state.TotalOrdering {
			points := make([]Point, 0)
			for k := range state.TotalOrdering[j] {
				points = append(points, state.Points[state.TotalOrdering[j][k]])
			}
			mergedPoints[i][j] = mergePoints(points)
			logger.Printf("Merged points id :%s\n\n===========\n", mergedPoints[i][j].Id)
		}
	}
	fmt.Println()
	return mergedPoints
}

func isFullCut(points []Point) bool {
	for _, p := range points {
		if p.Id == "" {
			return false
		}
	}
	return true
}

func noMerge(states []State) [][]Point {
	mergedPoints := make([][]Point, len(states))
	fmt.Println("No Merging points")
	return mergedPoints
}

func singleCutMerge(states []State) [][]Point {
	mergedPoints := make([][]Point, len(states))
	for i, state := range states {
		fmt.Printf("\rMerging States %3.0f%%", float32(i)/float32(len(states))*100)
		if !isFullCut(state.Points) {
			continue
		}
		mergedPoints[i] = make([]Point, 1)
		sort.Sort(ById(state.Points))
		// fmt.Printf("SCM: len of state.Points: %d", len(state.Points))
		mergedPoints[i][0] = mergePoints(state.Points)
	}
	logger.Printf("len of mergedPoints: %d", len(mergedPoints))
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
		//logger.Printf("id:%s\n", point.Id)
		mergedPoint.Id = mergedPoint.Id + "_" + point.Id
		pVClock1, _ := vclock.FromBytes(mergedPoint.VectorClock)
		pVClock2, _ := vclock.FromBytes(point.VectorClock)
		temp := pVClock1.Copy()
		temp.Merge(pVClock2)
		mergedPoint.VectorClock = temp.Bytes()
	}
	return mergedPoint
}

func ThreadCount(size int) int {
	var numCPU = runtime.NumCPU()
	var div int
	//dont bother splitting if it's not worth it
	if size < THREAD_BOOST {
		div = 1
	} else {
		div = numCPU
	}
	return div
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func writeShiVizLog(pointLog [][]Point, goLogs []*golog) {
	file, _ := os.Create("Shiviz.log")
	shivizRegex := "(?<host>\\S*) (?<clock>{.*})\\n(?<event>.*)\\n(?<dump>.*)"
	file.WriteString(shivizRegex)
	file.WriteString("\n\n")
	//TODO add in information about dump statements to logs will look
	//something like the (matchSendAndReceive) function in utils,
	//involving the backtracking through duplicate clock valued events
	for i, goLog := range goLogs {
		for j := range goLog.clocks {
			var dumpString string
			points := getEventsWithIdenticalHostTime(pointLog[i], goLog.id, j)
			for _, point := range points {
				dumpString += point.String()
			}
			log := fmt.Sprintf("%s %s\n%s\n%s\n", goLog.id, goLog.clocks[j].ReturnVCString(), goLog.messages[j], dumpString)
			file.WriteString(log)
		}
	}
}

func writeProgress(output string) {
	fmt.Printf("\r%s", CLEAR_LINE)
	fmt.Printf("\r%s", output)
}
