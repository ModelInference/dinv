// LogMerger
package logmerger

import (
	"encoding/gob"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"bitbucket.org/bestchai/dinv/govec/vclock"
	"gopkg.in/eapache/queue.v1"
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
	merged := mergeLogs(logs)
	writeLogToFile(merged)
}

//
func buildLogs(logFiles []string) [][]Point {
	logs := make([][]Point, 0)
	for i := 0; i < len(logFiles); i++ {
		print(i)
		log := readLog(logFiles[i])
		log = addBaseLog(log)
		name := fmt.Sprintf("L%d.", i)
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
			//fmt.Println(dp.VarName)
			logs[i].Dump[j].VarName = name + logs[i].Dump[j].VarName
		}
	}
}

//writeLogToFile produces a daikon dtrace file based on a log
//represented as an array of points
func writeLogToFile(log []Point) {

	file, _ := os.Create("daikonLog.dtrace")
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

//deprecated log merger
func merge2Logs(log1, log2 []Point) []Point {

	mergedPoints := make([]Point, 0)
	for i := 0; i < len(log1); i++ {

		matchedPoints := findMatch(log1[i], log2)
		fmt.Println(matchedPoints)
		for j := 0; j < len(matchedPoints); j++ {
			mergedPoints = append(mergedPoints, mergePoints([]Point{matchedPoints[j], log1[i]}))
		}
	}

	return mergedPoints
}

func ConsistantCuts(logs [][]Point) int {
	clocks, _ := VectorClockArraysFromLogs(logs)
	lattice := BuildLattice(clocks)
	printLattice(lattice)
	logs = enumerateCommunication(logs)
	consistentCuts := mineConsistentCuts(lattice, logs)
	for i := range consistentCuts {
		fmt.Println(consistentCuts[i].String())
	}
	return 0
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

//getLogId returns the first entry in the vector clock assuming that to be the owner
//TODO this is not that robust and takes advantage of the fact the logs have not been sorted
func getClockId(clocks []vclock.VClock) string {
	clock := clocks[0]
	re := regexp.MustCompile("{\"([A-Za-z0-9]+)\"")
	vString := clock.ReturnVCString()
	match := re.FindStringSubmatch(vString)
	return match[1]
}

func idClockMapper(clocks [][]vclock.VClock) []string {
	clockMap := make([]string, 0)
	for _, clock := range clocks {
		id := getClockId(clock)
		clockMap = append(clockMap, id)
	}
	return clockMap
}

func BuildLattice(clocks [][]vclock.VClock) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	for i := range clocks {
		latticePoint.Update(ids[i], 0)
		print(ids[i])
	}
	lattice := make([][]vclock.VClock, 0)
	current := queue.New()
	next := queue.New()
	next.Add(latticePoint)
	for next.Length() > 0 {
		lattice = append(lattice, make([]vclock.VClock, 0))
		current = next
		next = queue.New()
		for current.Length() > 0 {
			p := current.Peek().(*vclock.VClock)
			current.Remove()
			for i := range ids {
				pu := p.Copy()
				pu.Update(ids[i], 0)
				if !queueContainsClock(next, pu) && correctLatticePoint(clocks[i], pu, ids[i]) {
					//pu.PrintVC()
					next.Add(pu)
				}
			}
			lattice[len(lattice)-1] = append(lattice[len(lattice)-1], *p.Copy())
		}
	}
	return lattice
}

func queueContainsClock(q *queue.Queue, v *vclock.VClock) bool {
	for i := 0; i < q.Length(); i++ {
		check := q.Get(i).(*vclock.VClock)
		if v.Compare(check, vclock.Equal) {
			return true
		}
	}
	return false
}

func correctLatticePoint(clocks []vclock.VClock, proposedClock *vclock.VClock, id string) bool {
	found, index := searchClockById(clocks, proposedClock, id)
	//if the exact value was not found, then it was a non logged local
	//event, in this case the vector clock previous to the recieve is
	//used
	if !found {
		index, found = nearestPrecedingClock(clocks, proposedClock, index, id)
	}
	foundClock := clocks[index]
	if foundClock.HappenedBefore(proposedClock) && found {
		return true
	}
	return false
}

//searchLogForClock searches the log file for a clock value in key
//clock with the specified id
//if such an index is found, the index is returned with a matching
//true value, if no such index is found, the closest index is returned
//with a false value
func searchClockById(clocks []vclock.VClock, keyClock *vclock.VClock, id string) (bool, int) {
	min, max, mid := 0, len(clocks)-1, 0
	for max >= min {
		mid = min + ((max - min) / 2)
		a, _ := clocks[mid].FindTicks(id)
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

//nearestPrecedingClocks returns the closest preceding clock to
//proposed the proposed clock if the searched for
//index searched for did not return an exact matching timestamp.
//bestAttempt returns false if the index is out of bounds
//if the indexed log happened before the proposed value, that log is
//used
//if the indexed log happened after, the most recent preceding index
//is returned
func nearestPrecedingClock(clocks []vclock.VClock, proposedClock *vclock.VClock, index int, id string) (int, bool) {
	loggedTicks, _ := clocks[index].FindTicks(id)
	proposedTicks, _ := proposedClock.FindTicks(id)
	if index == 0 && proposedTicks < loggedTicks {
		print("out of bounds")
		return 0, false
	} else if index >= len(clocks)-1 && proposedTicks > loggedTicks {
		return 0, false
	} else if proposedTicks > loggedTicks {
		return index, true
	} else {
		return index - 1, true
	}
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
				fmt.Printf("consistent: %s\n", potentialCut.String())

			} else {
				fmt.Printf("inconsistent: %s\n", potentialCut.String())
			}
		}
	}
	return consistentCuts
}

func printLattice(lattice [][]vclock.VClock) {
	for i := range lattice {
		for j := range lattice[i] {
			v := lattice[i][j].ReturnVCString()
			fmt.Print(v)
		}
		fmt.Println()
	}
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
		fmt.Printf("searching logs of %s\n", ids[i])
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

//matchSendAndRecieve find a corresponding recieve event based on a
//proposed sending vectorclock, if no such recive event can be found
//in the corresponding clocks, then matched is returned false,
//otherwise the receiver and receiver event correspond to the index in
//clocks where the receive occured
func matchSendAndReceive(sender vclock.VClock, clocks [][]vclock.VClock, senderId string) (receiver int, receiverEvent int, matched bool) {
	matched = false
	var receiveClock = vclock.New()
	for k := range clocks {
		if getClockId(clocks[k]) != senderId {
			found, index := searchClockById(clocks[k], &sender, senderId)
			if found {
				foundClock := clocks[k][index]
				//backtrack for earliest clock
				//TODO this is ugly make it better
				for index > 0 {
					lesserClock := clocks[k][index-1]
					lesserTicks, _ := lesserClock.FindTicks(senderId)
					foundTicks, _ := foundClock.FindTicks(senderId)
					if foundTicks == lesserTicks {
						foundClock = *lesserClock.Copy()
						index--
					} else {
						break
					}
				}
				if receiver < 0 || receiveClock.Compare(&foundClock, vclock.Ancestor) {
					receiveClock = foundClock.Copy()
					receiver, receiverEvent, matched = k, index, true
				}
			}
		}
	}
	return receiver, receiverEvent, matched
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

func mergeLogs(logs [][]Point) []Point {
	cuts := carveLogs(logs)
	return cuts
}

func carveLogs(logs [][]Point) []Point {
	subV := vclock.New()
	logMap := idLogMapper(logs)
	cuts := make([]Point, 0)
	cutFound := true
	for cutFound {
		subV, cutFound = minJoinClock(logs, subV)
		cut := getCut(subV, logs, logMap)
		cuts = append(cuts, cut)
		//create cuts from minimum vector clock
	}
	return cuts
}

func getCut(subV *vclock.VClock, logs [][]Point, logMap map[int]string) Point {
	fmt.Println("getting Cut")
	subV.PrintVC()
	cutPoints := make([]Point, 0)
	for i, log := range logs {
		id := logMap[i]
		ticks, _ := subV.FindTicks(id)
		fmt.Printf("searching log :id %s, ticks %d\n", id, ticks)
		exp := fmt.Sprintf("\"%s\":([0-9]+)", id)
		re := regexp.MustCompile(exp)
		for _, point := range log {
			vc, _ := vclock.FromBytes(point.VectorClock)
			vString := vc.ReturnVCString()
			match := re.FindStringSubmatch(vString)
			vector, _ := strconv.Atoi(match[1])
			//fmt.Printf("Check :%s\n", vString)
			if vector == int(ticks) {
				fmt.Printf("Match :%s\n", vString)
				cutPoints = append(cutPoints, point)
				break
			}
		}
	}
	cut := mergePoints(cutPoints)
	return cut

}

//minJoinClock searches through a set of logs searching for the first point at which all nodes
//have communicated past the poinst specified by subV. That clock value is returned.
func minJoinClock(logs [][]Point, subV *vclock.VClock) (minClock *vclock.VClock, found bool) {
	cutFound := false
	minG := 0
	minV := vclock.New()
	for _, log := range logs {
		for _, point := range log {
			vp, _ := vclock.FromBytes(point.VectorClock)
			localMin, pnodes := subV.Difference(vp)
			//a new min cut is found if it involvs all nodes, and has the smallest vclock value
			if (localMin < minG || minG == 0) && pnodes == len(logs) {
				minV = vp.Copy()
				minG = localMin
				cutFound = true
			}
		}
	}
	return minV, cutFound
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
		mergedPoint.LineNumber = mergedPoint.LineNumber + "-" + point.LineNumber
		pVClock1, err := vclock.FromBytes(mergedPoint.VectorClock)
		printErr(err)
		pVClock2, err := vclock.FromBytes(point.VectorClock)
		temp := pVClock1.Copy()
		printErr(err)
		temp.Merge(pVClock2)
		mergedPoint.VectorClock = temp.Bytes()
	}
	return mergedPoint
}

// findMatch matches the vector clock at point with all points in log
// the set of points with matching vector clocks are returned
func findMatch(point Point, log []Point) []Point {
	matched := make([]Point, 0)
	pVClock, err := vclock.FromBytes(point.VectorClock)
	//fmt.Println(pVClock)
	printErr(err)
	for i := 0; i < len(log); i++ {

		otherVClock, err2 := vclock.FromBytes(log[i].VectorClock)
		printErr(err2)

		if pVClock.Matches(otherVClock) {
			matched = append(matched, log[i])
		}
	}

	return matched
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

type Cut struct {
	Points []Point
}

func (c Cut) String() string {
	catString := fmt.Sprintf("{")
	for i := range c.Points {
		vc, _ := vclock.FromBytes(c.Points[i].VectorClock)
		catString = fmt.Sprintf("%s | (VC: %s, CD: %d", catString, vc.ReturnVCString(), c.Points[i].CommunicationDelta)
	}
	catString = fmt.Sprintf("%s}", catString)
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
