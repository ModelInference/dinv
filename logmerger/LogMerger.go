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
	lattice := GenerateLattice(logs)
	printLattice(lattice)
	return 0
}

func GenerateLattice(logs [][]Point) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idLogMapper(logs)
	for i := range logs {
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
				if !queueContainsClock(next, pu) && validLatticePoint(logs[i], pu, ids[i]) {
					//pu.PrintVC()
					next.Add(pu)
				}
			}
			lattice[len(lattice)-1] = append(lattice[len(lattice)-1], *p.Copy())
		}
	}
	return lattice
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

func validLatticePoint(log []Point, proposedClock *vclock.VClock, id string) bool {
	found, index := searchLogForClock(log, proposedClock, id)
	//if the exact value was not found, then it was a non logged local
	//event, in this case the vector clock previous to the recieve is
	//used
	if !found {
		index, found = bestAttemptLog(log, proposedClock, index, id)
	}
	foundClock, _ := vclock.FromBytes(log[index].VectorClock)
	if foundClock.HappenedBefore(proposedClock) && found {
		return true
	}
	return false
}

//searchLogForClock searches the log file for a clock value in key
//clock with the specified id
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

//bestAttemptLog tries to find a logging point if the searched for
//index searched for did not return an exact matching timestamp.
//bestAttempt returns false if the index is out of bounds
//if the indexed log happened before the proposed value, that log is
//used
//if the indexed log happened after, the most recent preceding index
//is returned
func bestAttemptLog(log []Point, proposedClock *vclock.VClock, index int, id string) (int, bool) {
	loggedClock, _ := vclock.FromBytes(log[index].VectorClock)
	loggedTicks, _ := loggedClock.FindTicks(id)
	proposedTicks, _ := proposedClock.FindTicks(id)
	if index == 0 && proposedTicks < loggedTicks {
		print("out of bounds")
		return 0, false
	} else if index >= len(log)-1 && proposedTicks > loggedTicks {
		return 0, false
	} else if proposedTicks > loggedTicks {
		return index, true
	} else {
		return index - 1, true
	}
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

type Point struct {
	Dump        []NameValuePair
	LineNumber  string
	VectorClock []byte
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
