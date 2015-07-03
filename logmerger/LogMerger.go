// LogMerger
package logmerger

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"bitbucket.org/bestchai/dinv/govec/vclock"
	//"reflect"
)

var usage = "logmerger log1.txt log2.txt"
var debug = false
var verbose = true

func Merge(logfiles []string) {
	logs := buildLogs(logfiles)
	states := mineStates(logs)
	spec := &MergeSpec{totalOrderLineNumberMerge, 100, false}
	writeTraceFiles(states, spec)
}

func buildLogs(logFiles []string) [][]Point {
	logs := make([][]Point, 0)
	for i := 0; i < len(logFiles); i++ {
		log := readLog(logFiles[i])
		logs = append(logs, log)
	}
	clocks, _ := VectorClockArraysFromLogs(logs)
	ids := idClockMapper(clocks)
	for i, log := range logs {
		log = addBaseLog(ids[i], log)
		addNodeName(ids[i], log)
	}
	return logs
}

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

func createMapOfLogsForEachPoint(log []Point) map[string][]Point {
	mapOfPoints := make(map[string][]Point, 0)
	for i := 0; i < len(log); i++ {
		mapOfPoints[log[i].LineNumber] = append(mapOfPoints[log[i].LineNumber], log[i])
	}
	return mapOfPoints
}

func writeDeclaration(file *os.File, mapOfPoints map[string][]Point) {

	file.WriteString("decl-version 2.0\n")
	file.WriteString("var-comparability none\n")
	file.WriteString("\n")
	for _, v := range mapOfPoints {
		point := v[0]
		file.WriteString(fmt.Sprintf("ppt p-%s:::%s\n", point.LineNumber, point.LineNumber))
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

func writeValues(file *os.File, log []Point) {
	for i := range log {
		point := log[i]
		file.WriteString(fmt.Sprintf("p-%s:::%s\n", point.LineNumber, point.LineNumber))
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

func mineStates(logs [][]Point) []State {
	if verbose {
		fmt.Printf("\nStripping Clocks... ")
	}
	clocks, _ := VectorClockArraysFromLogs(logs)
	if verbose {
		fmt.Printf("Done\nBuilding Lattice... ")
	}
	lattice := BuildLattice(clocks)
	if verbose {
		fmt.Printf("Done\nCalculating Delta... ")
	}
	deltaComm := enumerateCommunication(clocks)
	if verbose {
		fmt.Printf("Done\nMining Consistent Cuts... ")
	}
	consistentCuts := mineConsistentCuts(lattice, clocks, deltaComm)
	if verbose {
		fmt.Printf("Done\nExtracting States... ")
	}
	states := statesFromCuts(consistentCuts, clocks, logs)
	if verbose {
		fmt.Printf("Done\n")
	}
	return states
}

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
		if debug {
			fmt.Printf("%s\n", state.String())
		}
		states = append(states, *state)
	}
	return states
}

func writeTraceFiles(states []State, spec *MergeSpec) {
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
						if debug {
							fmt.Printf("New file :%s\n", mergedPoints[i][j].LineNumber)
						}
						filename = mergedPoints[i][j].LineNumber
						newFile = true
					}
					if filename == mergedPoints[i][j].LineNumber {
						sample := rand.Int() % 100
						if sample < spec.sampleRate {
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
			if debug {
				fmt.Println("%s\n", mergedPoints[i][j].LineNumber)
			}
		}
	}
	return mergedPoints
}

func entireCutMerge(states []State) [][]Point {
	mergedPoints := make([][]Point, len(states))
	for i, state := range states {
		mergedPoints[i] = make([]Point, 1)
		mergedPoints[i][0] = mergePoints(state.Points)
	}
	return mergedPoints
}

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
		mergedPoint.LineNumber = mergedPoint.LineNumber + "_" + point.LineNumber
		pVClock1, _ := vclock.FromBytes(mergedPoint.VectorClock)
		pVClock2, _ := vclock.FromBytes(point.VectorClock)
		temp := pVClock1.Copy()
		temp.Merge(pVClock2)
		mergedPoint.VectorClock = temp.Bytes()
	}
	return mergedPoint
}

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

type MergeSpec struct {
	merger             func(state []State) [][]Point
	sampleRate         int
	totallyOrderedCuts bool
}

type Point struct {
	Dump               []NameValuePair
	LineNumber         string
	VectorClock        []byte
	CommunicationDelta int
}

type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}

func (nvp NameValuePair) String() string {
	return fmt.Sprintf("(%s,%s,%s)", nvp.VarName, nvp.Value, nvp.Type)
}

func (p Point) String() string {
	return fmt.Sprintf("%d : %s", p.LineNumber, p.Dump)
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
