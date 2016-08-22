/*
Logging distributed programs with vector clocks provides a partial
ordering of events across all participating hosts. The exact order of
events cannot be determined, however the clocks provide bounds to the
number of possible event orderings. Using the clocks as boundries all
potential event orderings can be calculated. The set of all such
orderings can be represented as a lattice, where each verticie
represents a potential state across all hosts in the distreibuted
program. This package generates such a lattice based on a set of
vector clocks

Author: Stewart Grant
Edited: July 6 2015
*/

package logmerger

import (
	"fmt"

	"github.com/arcaneiceman/GoVector/govec/vclock"
	"gopkg.in/eapache/queue.v1"
)

//BuildLattice constructs a lattice based on an ordered set of vector clocks. The
//computed lattice is represented as a 2-D array of vector clocks
//[i][j] where the i is the level of the lattice, and j is a set of
//events at that level
func BuildLattice2(clocks [][]vclock.VClock) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	levels := sumTime(clocks)
	for i := range clocks {
		latticePoint.Update(ids[i], 0)
	}
	level, points := 0, 0
	lattice := make([][]vclock.VClock, 0)
	current, next := queue.New(), queue.New()
	next.Add(latticePoint)
	for next.Length() > 0 {
		lattice = append(lattice, make([]vclock.VClock, 0))
		current = next
		nextMap := make(map[string]*vclock.VClock)
		for current.Length() > 0 {
			p := current.Peek().(*vclock.VClock)
			current.Remove()
			for i := range ids {
				pu := p.Copy()
				pu.Update(ids[i], 0)
				vstring := pu.ReturnVCString()
				_, ok := nextMap[vstring]
				if !ok && correctLatticePoint(clocks[i], pu, ids[i]) {
					//fmt.Println(vstring)
					nextMap[vstring] = pu
					points++
				}
			}
			next = mapToQueue(nextMap)
			fmt.Printf("\rComputing lattice  %3.0f%% \t point %d\t fanout %d", 100*float32(level)/float32(levels), points, len(lattice[level]))
			lattice[len(lattice)-1] = append(lattice[len(lattice)-1], *p.Copy())
		}
		levelToString(&lattice[level])
		level++
	}
	fmt.Println()
	return lattice

}

//BuildLattice constructs a lattice based on an ordered set of vector clocks. The
//computed lattice is represented as a 2-D array of vector clocks
//[i][j] where the i is the level of the lattice, and j is a set of
//events at that level
func BuildLattice3(clocks [][]vclock.VClock) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	mClocks := clocksToMaps(clocks) //clockWrapper
	levels := sumTime(clocks)

	// add a common starting point to the lattice:
	// for every host (the first level of the clocks array) a new clock tick at time 0 is inserted
	for i := range clocks {
		latticePoint.Update(ids[i], 0)
	}

	// 'points' and 'levelPoints' are only needed for progress output, not for the algorithm itself
	level, points := 0, 0
	// make the total lattice the number of levels squared for safty equal to the size of the ids
	lattice := make([][]vclock.VClock, levels*levels)
	lattice[level] = make([]vclock.VClock, 1)
	lattice[level][0] = *latticePoint

	// iterate over the levels, starting with level 1 as long as the lattice still has nodes on the next level
	for len(lattice[level]) > 0 {
		level++
		// next lattice level = previous *num nodes (worst case)
		nextMap := make(map[string]*vclock.VClock, len(lattice[level-1])*len(ids))
		levelPoints := 0
		for j := range lattice[level-1] {
			p := lattice[level-1][j]
			//fmt.Printf("Inital Point %s\n",p.ReturnVCString())
			for i, id := range ids {
				pu := p.Copy()
				pu.Update(ids[i], 0)
				vstring := pu.ReturnVCString()
				_, ok := nextMap[vstring]
				if !ok && fastCorrectLatticePoint(mClocks, pu, id) {
					//fmt.Println(vstring)
					nextMap[vstring] = pu
					levelPoints++
				}
			}
		}
		lattice[level] = mapToArray(nextMap)
		fmt.Printf("\rComputing lattice  %3.0f%% \t points %d\t fanout %d", 100*float32(level)/float32(levels), points, len(lattice[level]))
		//levelToString(&lattice[level])
		points += levelPoints
	}
	fmt.Println()
	return lattice
}

func CompareLattice(a, b [][]vclock.VClock) {
	for i := range a {
		for j := range a[i] {
			found := false
			for k := range b[i] {
				if a[i][j].Compare(&b[i][k], vclock.Equal) {
					found = true
				}
			}
			if !found {
				fmt.Printf("unequal lattice point %s\n", a[i][j].ReturnVCString())
			}
		}
	}
}

func levelToString(level *[]vclock.VClock) {
	fmt.Println("-----------------------------------------------")
	for i := range *level {
		fmt.Printf("##%s##", (*level)[i].ReturnVCString())
	}
	fmt.Println("-----------------------------------------------")
}

func mapToArray(vmap map[string]*vclock.VClock) []vclock.VClock {
	array := make([]vclock.VClock, len(vmap))
	i := 0
	for _, clock := range vmap {
		array[i] = *clock
		i++
	}
	return array[0:i]
}

func mapToQueue(vmap map[string]*vclock.VClock) *queue.Queue {
	q := queue.New()
	for _, clock := range vmap {
		q.Add(clock)
	}
	return q
}

//return id -> clockValue -> vectorClock map
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

func PrintMaps(clocks map[string]map[uint64]map[string]uint64) {
	for id := range clocks {
		fmt.Println(id)
		i := uint64(1)
		for range clocks[id] {
			fmt.Print("[")
			for cid := range clocks[id][i] {
				fmt.Printf(" %s : %d ", cid, clocks[id][i][cid])
			}
			fmt.Printf("]\n")
			i++
		}
	}
}

//BuildLattice constructs a lattice based on an ordered set of vector clocks. The
//computed lattice is represented as a 2-D array of vector clocks
//[i][j] where the i is the level of the lattice, and j is a set of
//events at that level
func BuildLattice(clocks [][]vclock.VClock) [][]vclock.VClock {
	fmt.Println("lattices are cool")
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	levels := sumTime(clocks)
	for i := range clocks {
		latticePoint.Update(ids[i], 0)
	}
	level, points := 0, 0
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
					points++
				}
			}
			fmt.Printf("\rComputing lattice  %3.0f%% \t point %d\t fanout %d", 100*float32(level)/float32(levels), points, len(lattice[level]))
			lattice[len(lattice)-1] = append(lattice[len(lattice)-1], *p.Copy())
		}
		level++
	}
	fmt.Println()
	return lattice
}

//queueContainsClock searches a queue of unorded vector clocks q for a
//match to the argument v. If v is in the queue return true, otherwise
//false
func queueContainsClock(q *queue.Queue, v *vclock.VClock) bool {
	for i := 0; i < q.Length(); i++ {
		check := q.Get(i).(*vclock.VClock)
		if v.Compare(check, vclock.Equal) {
			return true
		}
	}
	return false
}

//correctLatticePoint searches an array of clocks, belonging to a
//host specificed by id. The proposed clock is a set of clock values
//which could have potentially happened with respect to the host. If
//the set is possible return true, otherwise false.
func correctLatticePoint(loggedClocks []vclock.VClock, latticePoint *vclock.VClock, id string) bool {
	found, index := searchClockById(loggedClocks, latticePoint, id)
	//if the exact value was not found, then it was a non logged local
	//event, in this case the vector clock previous to the recieve is
	//used
	if !found {
		index, found = nearestPrecedingClock(loggedClocks, latticePoint, index, id)
	}
	hostClock := loggedClocks[index]
	if LatticeValidWithLog(latticePoint, &hostClock) && found {
		return true
	}
	return false
}

func fastCorrectLatticePoint(mClocks map[string]map[uint64]map[string]uint64, point *vclock.VClock, id string) bool {
	//fmt.Printf("id match = %s\n",id)
	clockValue, found := point.FindTicks(id)
	clock, ok := mClocks[id][clockValue]
	if !ok {
		logger.Printf("Log does not contain a clock for %s val: %d\n", id, clockValue)
		return false
	}
	if !found {
		logger.Printf("id %s not found in point\n", id)
		return false
	}
	if clock[id] != clockValue {
		logger.Printf("o shit what are you doing! %d != %d\n", clock[id], clockValue)
		return false
	}
	//fmt.Printf("lattice Clock: %s len logged :%d\n",point.ReturnVCString(),len(clock))
	for cid, loggedTicks := range clock {
		latticeTicks, _ := point.FindTicks(cid)
		//fmt.Printf("p id:%s  ticks: %d \n",cid,loggedTicks)
		if loggedTicks > latticeTicks {
			//fmt.Printf("Rejected lattice Clock: %s\n",point.ReturnVCString())
			return false
			//fmt.Println()
		}
		//fmt.Println()
	}
	//fmt.Println("added")
	return true
}

func LatticeValidWithLog(latticePoint, loggedClock *vclock.VClock) bool {
	latticeIds := getClockIds(latticePoint)
	for _, id := range latticeIds {
		loggedTicks, _ := loggedClock.FindTicks(id)
		latticeTicks, _ := latticePoint.FindTicks(id)
		if loggedTicks > latticeTicks {
			return false
		}
	}
	return true
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

//PrintLattice itterates through each level of the lattice structure
//and prints out the clocks at each level
func PrintLattice(lattice [][]vclock.VClock) {
	for i := range lattice {
		for j := range lattice[i] {
			v := lattice[i][j].ReturnVCString()
			fmt.Print(v)
		}
		fmt.Println()
	}
}
