package logmerger

import (
	"fmt"
	"regexp"

	"bitbucket.org/bestchai/dinv/govec/vclock"
	"gopkg.in/eapache/queue.v1"
)

func BuildLattice(clocks [][]vclock.VClock) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	for i := range clocks {
		latticePoint.Update(ids[i], 0)
		print(ids[i])
	}
	lattice := make([][]vclock.VClock, 0)
	current, next := queue.New(), queue.New()
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
					next.Add(pu)
				}
			}
			lattice[len(lattice)-1] = append(lattice[len(lattice)-1], *p.Copy())
		}
	}
	return lattice
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

func idClockMapper(clocks [][]vclock.VClock) []string {
	clockMap := make([]string, 0)
	for _, clock := range clocks {
		id := getClockId(clock)
		clockMap = append(clockMap, id)
	}
	return clockMap
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

//getLogId returns the first entry in the vector clock assuming that to be the owner
//TODO this is not that robust and takes advantage of the fact the logs have not been sorted
func getClockId(clocks []vclock.VClock) string {
	clock := clocks[0]
	re := regexp.MustCompile("{\"([A-Za-z0-9]+)\"")
	vString := clock.ReturnVCString()
	match := re.FindStringSubmatch(vString)
	return match[1]
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
