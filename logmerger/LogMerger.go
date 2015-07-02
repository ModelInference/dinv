// LogMerger
package logmerger

import (
	"encoding/gob"
	"fmt"
	"os"
	"regexp"
	"strings"

	"bitbucket.org/bestchai/dinv/govec/vclock"
	//"reflect"
)

var usage = "logmerger log1.txt log2.txt"
var debug = true

/*
func main() {
}
*/

func Merge(logfiles []string) {
	logs := buildLogs(logfiles)
	states := mineStates(logs)
	writeTraceFiles(states)
}

func buildLogs(logFiles []string) [][]Point {
	logs := make([][]Point, 0)
	for i := 0; i < len(logFiles); i++ {
		print(i)
		log := readLog(logFiles[i])
		log = addBaseLog(log)
		name := fmt.Sprintf("L%d.", i)
		fmt.Println(name)
		addNodeName(name, log)
		logs = append(logs, log)
	}
	return logs
}

func addBaseLog(log []Point) []Point {
	name := getLogId(log)
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
			logs[i].Dump[j].VarName = name + logs[i].Dump[j].VarName
			fmt.Println(logs[i].Dump[j].VarName)
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
	clocks, _ := VectorClockArraysFromLogs(logs)
	lattice := BuildLattice(clocks)
	//printLattice(lattice)
	logs = enumerateCommunication(logs)
	consistentCuts := mineConsistentCuts(lattice, logs)
	states := statesFromCuts(consistentCuts, clocks)
	/*for i := range states {
		fmt.Println(states[i].String())
	}*/
	return states
}

func countAncestors(cut Cut) []int {
	ancestors := make([]int, len(cut.Points))
	for i := range cut.Points {
		for j := range cut.Points {
			if i != j {
				clock1, _ := vclock.FromBytes(cut.Points[i].VectorClock)
				clock2, _ := vclock.FromBytes(cut.Points[j].VectorClock)
				if clock1.Compare(clock2, vclock.Ancestor) {
					ancestors[i]++
				}
			}
		}
	}
	return ancestors
}

func totalOrderFromCut(cut Cut, clocks [][]vclock.VClock) [][]int {
	used := make([]bool, len(cut.Points))
	ancestors := countAncestors(cut)
	ids := idClockMapper(clocks)
	ordering := make([][]int, 0)
	extracted := true
	for extracted {
		extracted = false
		//get oldest clock
		max, index := -1, -1
		for i := range ancestors {
			if ancestors[i] > max && !used[i] {
				max, index = ancestors[i], i
			}
		}
		if max < 0 {
			return ordering
		}
		ordering = append(ordering, make([]int, 0))
		ordering[len(ordering)-1] = append(ordering[len(ordering)-1], index)
		used[index] = true
		extracted = true

		child := true
		//TODO fix the base case for some reason this is making pairs
		//where it should not be
		for child {
			child = false
			//rclock, _ := vclock.FromBytes(cut.Points[index].VectorClock)
			maxEvent, sendIndex := -1, -1
			for i := range cut.Points {
				if i != index && !used[i] {
					sclock, _ := vclock.FromBytes(cut.Points[i].VectorClock)
					receiver, event, found := matchSendAndReceive(*sclock, clocks, ids[i])
					if found && receiver == index && event > maxEvent {
						maxEvent, sendIndex = event, i
					}
				}
			}
			if maxEvent >= 0 {
				ordering[len(ordering)-1] = append(ordering[len(ordering)-1], sendIndex)
				used[sendIndex] = true
				child = true
				index = sendIndex
			}
		}
	}
	return ordering
}

func statesFromCuts(cuts []Cut, clocks [][]vclock.VClock) []State {
	states := make([]State, 0)
	for _, cut := range cuts {
		state := &State{}
		state.Cut = cut
		state.TotalOrdering = totalOrderFromCut(cut, clocks)
		for i := range state.TotalOrdering {
			points := make([]Point, 0)
			for j := range state.TotalOrdering[i] {
				points = append(points, state.Cut.Points[state.TotalOrdering[i][j]])
			}
			state.MergedPoints = append(state.MergedPoints, mergePoints(points))
		}
		states = append(states, *state)
	}
	return states
}

//VectorClockArraysFromLogs extracts the set of vector clocks
//corresponding to thoses in a set of logs
//log[i] will produce a corresponding array of vector clocks clocks[i]
//In the case where a vector clock cannot be extracted from a log an
//error is returned
func VectorClockArraysFromLogs(logs [][]Point) ([][]vclock.VClock, error) {
	clocks := make([][]vclock.VClock, 0)
	for i := range logs {
		clocks = append(clocks, make([]vclock.VClock, 0))
		for j := range logs[i] {
			vc, err := vclock.FromBytes(logs[i][j].VectorClock)
			if err != nil {
				return nil, err
			} else {
				clocks[i] = append(clocks[i], *vc)
			}
			if debug {
				fmt.Println(vc.ReturnVCString())
			}
		}
	}
	return clocks, nil
}

func mineConsistentCuts(lattice [][]vclock.VClock, logs [][]Point) []Cut {
	ids := idLogMapper(logs)
	consistentCuts := make([]Cut, 0)
	for i := range lattice {
		for j := range lattice[i] {
			communicationDelta := 0
			var potentialCut Cut
			for k := range ids {
				_, found := lattice[i][j].FindTicks(ids[k])
				if !found {
					break
				}
				found, index := searchLogForClock(logs[k], &lattice[i][j], ids[k])
				communicationDelta += logs[k][index].CommunicationDelta
				potentialCut.Points = append(potentialCut.Points, logs[k][index])
			}
			if communicationDelta == 0 {
				consistentCuts = append(consistentCuts, potentialCut)
			}
		}
	}
	return consistentCuts
}

//searchLogForClock searches the log file for a clock value in key
//clock with the specified id,
//searchLogForClock assumes that the clocks are ordered in ascendin
//value
//if such an index is found, the index is returned with a matching
//true value, if no such index is found, the closest index is returned
//with a false valuR
func searchLogForClock(log []Point, keyClock *vclock.VClock, id string) (bool, int) {
	min, max, mid := 0, len(log)-1, 0
	for max >= min {
		mid = min + ((max - min) / 2)
		searchClock, _ := vclock.FromBytes(log[mid].VectorClock)
		a, _ := searchClock.FindTicks(id)
		b, _ := keyClock.FindTicks(id)
		if a == b {
			return true, mid
		} else if a < b {
			min = mid + 1
		} else {
			max = mid - 1
		}
	}
	return false, mid
}

//enumerateCommunication searches all the logs for sends and receives
//and keeps track of how many have been done on each host
func enumerateCommunication(logs [][]Point) [][]Point {
	ids := idLogMapper(logs)
	clocks, _ := VectorClockArraysFromLogs(logs)
	for i := range logs {
		for j := range logs[i] {
			receiver, receiverEvent, matched := matchSendAndReceive(clocks[i][j], clocks, ids[i])
			if matched {
				logs[i][j].CommunicationDelta++
				logs[receiver][receiverEvent].CommunicationDelta--
				if debug {
					fmt.Printf("SR pair found %s, %s\n", clocks[i][j].ReturnVCString(), clocks[receiver][receiverEvent].ReturnVCString())
					fmt.Printf("Sender %s:%d ----> Receiver %s:%d\n", ids[i], logs[i][j].CommunicationDelta, ids[receiver], logs[receiver][receiverEvent].CommunicationDelta)
				}
			}
		}
	}
	logs = fillCommunicationDelta(logs)
	//fill in the blanks
	return logs
}

//fillCommunicationDelta markes the difference in sends and recieves that
//have occured on a particualr host at every point throughout there
//exectuion
// a host with 5 sends and 2 recieves will be given the delta = 3
// a host with 10 receives and 5 sends will be given the delta = -5
func fillCommunicationDelta(logs [][]Point) [][]Point {
	for i := range logs {
		fill := 0
		for j := range logs[i] {
			if logs[i][j].CommunicationDelta != 0 {
				temp := logs[i][j].CommunicationDelta
				logs[i][j].CommunicationDelta += fill
				fill += temp
			} else {
				logs[i][j].CommunicationDelta += fill
			}
		}
	}
	return logs
}

//matchSendAndRecieve find a corresponding recieve event based on a
//proposed sending vectorclock, if no such recive event can be found
//in the corresponding clocks, then matched is returned false,
//otherwise the receiver and receiver event correspond to the index in
//clocks where the receive occured
func matchSendAndReceive(sender vclock.VClock, clocks [][]vclock.VClock, senderId string) (receiver int, receiverEvent int, matched bool) {
	receiver, receiverEvent, matched = -1, -1, false
	var receiveClock = vclock.New()
	for i := range clocks {
		if getClockId(clocks[i]) != senderId {
			found, event := searchClockById(clocks[i], &sender, senderId)
			if found {
				//backtrack for earliest clock
				//TODO this is ugly make it better
				for event > 0 {
					currentTicks, _ := clocks[i][event].FindTicks(senderId)
					prevTicks, _ := clocks[i][event-1].FindTicks(senderId)
					if currentTicks == prevTicks {
						event--
					} else {
						break
					}
				}
				//uses partial evaluation for protection, dont switch
				if receiver < 0 || receiveClock.Compare(&clocks[i][event], vclock.Ancestor) {
					receiveClock = clocks[i][event].Copy()
					receiver, receiverEvent, matched = i, event, true
				}
			}
		}
	}
	return receiver, receiverEvent, matched
}

func writeTraceFiles(states []State) {
	written := make([][]bool, len(states))
	for i := range states {
		written[i] = make([]bool, len(states[i].MergedPoints))
	}
	newFile := true
	for newFile {
		newFile = false
		var filename string
		pointLog := make([]Point, 0)
		for i := range states {
			for j := range states[i].MergedPoints {
				if !written[i][j] {
					if !newFile {
						filename = states[i].MergedPoints[j].LineNumber
						newFile = true
					}
					if filename == states[i].MergedPoints[j].LineNumber {
						pointLog = append(pointLog, states[i].MergedPoints[j])
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

func idLogMapper(logs [][]Point) map[int]string {
	logMap := make(map[int]string)
	for i, log := range logs {
		id := getLogId(log)
		logMap[i] = id
	}
	return logMap
}

//getLogId returns the first entry in the vector clock assuming that to be the owner
//TODO this is not that robust and takes advantage of the fact the logs have not been sorted
func getLogId(log []Point) string {
	point := log[0]
	re := regexp.MustCompile("{\"([A-Za-z0-9]+)\"")
	vc, _ := vclock.FromBytes(point.VectorClock)
	vString := vc.ReturnVCString()
	match := re.FindStringSubmatch(vString)
	return match[1]
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
	printErr(err)

	fmt.Println("decoding " + filePath)
	decoder := gob.NewDecoder(fileR)

	pointArray := make([]Point, 0)

	var e error = nil
	for e == nil {
		var decodedPoint Point
		e = decoder.Decode(&decodedPoint)
		if e == nil {
			pointArray = append(pointArray, decodedPoint)
		} else {
			fmt.Println(e)
		}
	}

	return pointArray
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

type State struct {
	Cut           Cut
	TotalOrdering [][]int
	MergedPoints  []Point
}

type Cut struct {
	Points []Point
}

func (c Cut) String() string {
	catString := fmt.Sprintf("{")
	for i := range c.Points {
		vc, _ := vclock.FromBytes(c.Points[i].VectorClock)
		catString = fmt.Sprintf("%s | (P: %s)\n (VC: %s)\n, (CD: %d)\n\n", catString, c.Points[i].String(), vc.ReturnVCString(), c.Points[i].CommunicationDelta)
	}
	catString = fmt.Sprintf("%s}", catString)
	return catString
}

func (state State) String() string {
	catString := fmt.Sprintf("%s\n[", state.Cut.String())
	for i := range state.TotalOrdering {
		catString = fmt.Sprintf("%s[", catString)
		for j := range state.TotalOrdering[i] {
			catString = fmt.Sprintf("%s %d,", catString, state.TotalOrdering[i][j])
		}
		catString = fmt.Sprintf("%s]", catString)
	}
	catString = fmt.Sprintf("%s]", catString)
	return catString
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
