/*
The processes of merging logs requires a number of common functions. Searching arrays for particular clock values, matching sends and recives, and resolving the owner of a log occur throught the process. These utility functions are available here

Author: Stewart Grant
Edited: July 6 2015
*/

package logmerger

import (
	"regexp"

	"bitbucket.org/bestchai/dinv/govec/vclock"
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
				clocks[i] = append(clocks[i], *vc)
			}
			logger.Println(vc.ReturnVCString())
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
	re := regexp.MustCompile("{\"([A-Za-z0-9]+)\"")
	vString := clock.ReturnVCString()
	match := re.FindStringSubmatch(vString)
	//fmt.Printf(" Found %s\n", match[1])
	return match[1]
}
