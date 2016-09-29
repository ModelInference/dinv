/*
The processes of merging logs requires a number of common functions. Searching arrays for particular clock values, matching sends and recives, and resolving the owner of a log occur throught the process. These utility functions are available here

Author: Stewart Grant
Edited: July 6 2015
*/

package logmerger

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/arcaneiceman/GoVector/govec/vclock"
)

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
				clocks[i] = append(clocks[i], vc)
			}
			logger.Println(vc.ReturnVCString())
		}
	}
	return clocks, nil
}

func VectorClockArraysFromGoVectorLogs(clockLogs []*golog) ([][]vclock.VClock, error) {
	clocks := make([][]vclock.VClock, 0)
	for i := range clockLogs {
		clocks = append(clocks, make([]vclock.VClock, 0))
		for j := range clockLogs[i].clocks {
			clocks[i] = append(clocks[i], clockLogs[i].clocks[j])
			//logger.Println(clocks[i][j].ReturnVCString())
		}
	}
	return clocks, nil
}

//searchLogForClock searches the log file for a clock value in key
//clock with the specified id,
//searchLogForClock assumes that the clocks are ordered in ascendin
//value
//if such an index is found, the index is returned with a matching
//true value, if no such index is found, the closest index is returned
//with a false valuR
func searchLogForClock(log []Point, keyClock vclock.VClock, id string) (bool, int) {
	min, max, mid := 0, len(log)-1, 0
	for max >= min {
		mid = min + ((max - min) / 2)
		searchClock, _ := vclock.FromBytes(log[mid].VectorClock)
		a, _ := searchClock.FindTicks(id)
		b, _ := keyClock.FindTicks(id)
		fmt.Printf("%d-%d", a, b)
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

//getEventsWithIndenticalHostTime, collects logged program points
//which have the same hostID and returns them
func getEventsWithIdenticalHostTime(points []Point, hostId string, time int) []Point {
	matchingPoints := make([]Point, 0)
	foundPoint := false
	for i := range points {
		pointClock, _ := vclock.FromBytes(points[i].VectorClock)
		ticks, found := pointClock.FindTicks(hostId)
		if found && int(ticks) == time {
			matchingPoints = append(matchingPoints, points[i])
			foundPoint = true
		} else if foundPoint {
			return matchingPoints
		}
	}
	return matchingPoints
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
			//logger.Printf(" Clock ID : %s -> sender Id %s\n", getClockId(clocks[i]), senderId)
			found, event := searchClockById(clocks[i], sender, senderId)
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
				if receiver < 0 || receiveClock.Compare(clocks[i][event], vclock.Ancestor) {
					receiveClock = clocks[i][event].Copy()
					receiver, receiverEvent, matched = i, event, true
				}
			}
		}
	}
	return receiver, receiverEvent, matched
}

func pointsToMaps(points [][]Point) map[string]map[uint64]Point {
	pointMap := make(map[string]map[uint64]Point, len(points))
	for i := range points {
		byteClock, _ := vclock.FromBytes(points[i][0].VectorClock)
		id := getClockId([]vclock.VClock{byteClock})
		pointMap[id] = make(map[uint64]Point, len(points[i]))
		for j := range points[i] {
			inClock, _ := vclock.FromBytes(points[i][j].VectorClock)
			value, _ := inClock.FindTicks(id)
			pointMap[id][value] = points[i][j]
		}
	}
	return pointMap
}

//return id -> clockValue -> vectorClock map
//TODO clocks were updated to maps[string]uint this function can be
//simplified
func clocksToMaps(clocks [][]vclock.VClock) map[string]map[uint64]map[string]uint64 {
	ids := idClockMapper(clocks)
	mClocks := make(map[string]map[uint64]map[string]uint64, len(ids))
	for i, id := range ids {
		mClocks[id] = make(map[uint64]map[string]uint64, len(clocks[i]))
		for j, _ := range clocks[i] {
			selfIndex, foundSelf := clocks[i][j].FindTicks(id)
			if foundSelf {
				mClocks[id][selfIndex] = make(map[string]uint64, len(ids))
				for _, cid := range ids {
					value, found := clocks[i][j].FindTicks(cid)
					if found {
						//logger.Printf("id = %s, index %d, cid %s, val %d\n",id,selfIndex,cid,value)
						mClocks[id][selfIndex][cid] = value
					} else {
						//logger.Printf("not found %s %d %s %d\n",id,selfIndex,cid,value)
					}
				}
			} else {
				fmt.Printf("cound not find self %s - %d\n", id, j+1)
			}
		}
	}
	fmt.Println("Done Mapping Clocks")
	//PrintMaps(mClocks)
	return mClocks
}

//searchLogForClock searches the log file for a clock value in key
//clock with the specified id
//if such an index is found, the index is returned with a matching
//true value, if no such index is found, the closest index is returned
//with a false value
func searchClockById(clocks []vclock.VClock, keyClock vclock.VClock, id string) (bool, int) {
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

func fastSearchClockById(mClocks map[string]map[uint64]map[string]uint64, point vclock.VClock, id string) (map[string]uint64, bool) {
	//fmt.Printf("id match = %s\n",id)
	clockValue, found := point.FindTicks(id)
	clock, ok := mClocks[id][clockValue]
	if !ok {
		//logger.Printf("Log does not contain a clock for %s val: %d\n", id, clockValue)
		return nil, false
	}
	if !found {
		logger.Printf("id %s not found in point\n", id)
		return nil, false
	}
	if clock[id] != clockValue {
		logger.Printf("o shit what are you doing! %d != %d\n", clock[id], clockValue)
		return nil, false
	}
	return clock, ok
}

func sumTime(clockSet [][]vclock.VClock) uint64 {
	ids := idClockMapper(clockSet)
	maxMap := make(map[string]uint64, len(ids))
	for _, clocks := range clockSet {
		last := clocks[len(clocks)-1]
		for _, id := range ids {
			ticks, _ := last.FindTicks(id)
			if ticks > maxMap[id] {
				maxMap[id] = ticks
			}
		}
	}
	var total uint64
	for _, value := range maxMap {
		total += value
	}

	return total
}

func ClockFromString(clock, regex string) (vclock.VClock, error) {
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

//idCLockMapper returns a set of id strings corresponding to the
//owners of each array of vector clocks. ie if clocks[i] had the host id
//HOST, the returned [i]string = "HOST"
func idClockMapper(clocks [][]vclock.VClock) []string {
	clockMap := make([]string, 0)
	for _, clock := range clocks {
		id := getClockId(clock)
		clockMap = append(clockMap, id)
	}
	return clockMap
}

//getLogId returns the first entry in the vector clock assuming that to be the owner
//TODO this is not that robust and takes advantage of the fact the logs have not been sorted
//TODO document the expected format & place documentation on webpage
func getClockId(clocks []vclock.VClock) string {
	//fmt.Printf("Searching Host ...")
	if len(clocks) < 1 {
		return "anon"
	}
	clock := clocks[0]
	re := regexp.MustCompile("{\"([A-Za-z0-9_]+)\"")
	vString := clock.ReturnVCString()
	match := re.FindStringSubmatch(vString)
	//fmt.Printf(" Found %s\n", match[1])
	return match[1]
}

func ConstructVclock(ids []string, ticks []int) vclock.VClock {
	if len(ids) != len(ticks) {
		return nil
	}
	clock := vclock.New()
	for i := range ids {
		if ticks[i] < 0 {
			return nil
		}
		clock.Set(ids[i], uint64(ticks[i]))
	}
	return clock
}

func getClockIds(clock vclock.VClock) []string {
	ids := make([]string, len(clock))
	i := 0
	for id := range clock {
		ids[i] = id
		i++
	}
	return ids
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

func Hash(id string) string {
	h := sha1.New()
	io.WriteString(h, id)
	bytes := fmt.Sprintf("%x", h.Sum(nil))
	return strings.Trim(bytes, " ")
}
