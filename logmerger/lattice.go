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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/arcaneiceman/GoVector/govec/vclock"
	"gopkg.in/eapache/queue.v1"
)

const (
	THREAD_BOOST     = 1000
	POINT_DISK_LIMIT = 5000000
)

type LatticeWrapper struct {
	LatticeM      [][]vclock.VClock //main memory lattice
	LatticeD      []string          //wrapper for the disk
	CurrentFile   int
	Points        uint64
	LevelM        int
	LevelD        int
	LevelEstimate int
}

func New() *LatticeWrapper {
	return &LatticeWrapper{nil, nil, 0, 0, 0, 0, 0}

}

func (lw *LatticeWrapper) Delete() {
	for i := range lw.LatticeD {
		os.Remove(lw.LatticeD[i])
	}
}

//set the current level of the lattice back to the beginning
func (lw *LatticeWrapper) Beginning() {
	lw.LevelM = 0
	lw.LevelD = 0
	lw.CurrentFile = 0
	if len(lw.LatticeD) > 0 {
		lw.FetchDisk()
	}
}

func (lw *LatticeWrapper) Push(layer []vclock.VClock) {
	lw.LatticeM[lw.LevelM] = layer

	//disk writing phase
	if lw.Points > POINT_DISK_LIMIT*(uint64(len(lw.LatticeD)+1)) {
		lw.DiskDump()

		temp := make([]vclock.VClock, len(lw.LatticeM[lw.LevelM]))
		for i := range temp {
			temp[i] = lw.LatticeM[lw.LevelM][i]
		}
		lw.LatticeM = make([][]vclock.VClock, lw.LevelEstimate)

		lw.LatticeM[0] = make([]vclock.VClock, len(temp))
		for i := range temp {
			lw.LatticeM[0][i] = temp[i]
		}
		lw.LevelM = 0
		lw.CurrentFile++
	}
	lw.Points += uint64(len(lw.LatticeM[lw.LevelM]))
	lw.LevelM++
	lw.LevelD++

}

func (lw *LatticeWrapper) Pop() []vclock.VClock {
	//nothing left to pop
	if lw.CurrentFile == (len(lw.LatticeD)-1) && lw.LevelM == (len(lw.LatticeM)-1) {
		return nil
	} else if lw.LevelM == len(lw.LatticeM)-1 && len(lw.LatticeD) > 0 {
		//fetch from disk
		lw.CurrentFile++
		lw.FetchDisk()
		lw.LevelM = 0
	}
	ret := lw.LatticeM[lw.LevelM]
	lw.LevelM++
	lw.LevelD++
	return ret
}

func (lw *LatticeWrapper) FetchDisk() {
	fmt.Printf("%s", CLEAR_LINE)
	latticeFile, err := os.Open(lw.LatticeD[lw.CurrentFile])
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(latticeFile)
	lattice := make([][]vclock.VClock, 0)
	var e error = nil

	stat, _ := latticeFile.Stat()
	var soFar uint64
	size := stat.Size()
	for e == nil {
		var decodedLayer []vclock.VClock
		e = decoder.Decode(&decodedLayer)
		soFar += uint64(unsafe.Sizeof(&decodedLayer))
		// Will probably have to do -- fixJsonEncodingTypeConversion(&decodedPoint)
		if e == nil {
			lattice = append(lattice, decodedLayer)
		} else {
			fmt.Println(e)
		}
		writeProgress(fmt.Sprintf("Fetching lattice from Disk %3.0f%% ", 100*float64(soFar)/float64(size)))
	}
	latticeFile.Close()
	lw.LatticeM = lattice

}

func (lw *LatticeWrapper) DiskDump() {
	fmt.Printf("%s", CLEAR_LINE)
	latticeFilename := fmt.Sprintf("L%d", len(lw.LatticeD))
	latticeFile, err := os.Create(latticeFilename)
	if err != nil {
		panic(err)
	}
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	for i := 0; i < lw.LevelM-1; i++ {
		encoder.Encode(lw.LatticeM[i])
		latticeFile.Write(buf.Bytes())
		buf.Reset()
		writeProgress(fmt.Sprintf("Writing to Disk %3.0f%% ", 100*float32(i)/float32(lw.LevelM)))
	}
	err = latticeFile.Close()
	if err != nil {
		panic(err)
	}
	lw.LatticeD = append(lw.LatticeD, latticeFilename)
}

//BuildLattice constructs a lattice based on an ordered set of vector clocks. The
//computed lattice is represented as a 2-D array of vector clocks
//[i][j] where the i is the level of the lattice, and j is a set of
//events at that level
//Build lattice 4 is threaded
func BuildLattice5(clocks [][]vclock.VClock) *LatticeWrapper {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	mClocks := clocksToMaps(clocks) //clockWrapper
	levels := sumTime(clocks)

	// add a common starting point to the lattice:
	// for every host (the first level of the clocks array) a new clock tick at time 0 is inserted
	for i := range clocks {
		latticePoint.Tick(ids[i])
	}

	// 'points' and 'levelPoints' are only needed for progress output, not for the algorithm itself
	// make the total lattice the number of levels squared for safty equal to the size of the ids
	lw := New()
	lw.LevelEstimate = int(levels)
	lw.LatticeM = make([][]vclock.VClock, lw.LevelEstimate)
	lw.LatticeM[lw.LevelM] = make([]vclock.VClock, 1)
	l1 := make([]vclock.VClock, 1)
	l1[0] = latticePoint
	lw.LatticeM[lw.LevelM][0] = latticePoint
	lw.Push(l1)

	// iterate over the levels, starting with level 1 as long as the lattice still has nodes on the next level
	for len(lw.LatticeM[lw.LevelM-1]) > 0 {
		// next lattice level = previous *num nodes (worst case)
		nextMap := make(map[string]vclock.VClock, len(lw.LatticeM[lw.LevelM-1])*len(ids))
		levelPoints := 0

		div := ThreadCount(len(lw.LatticeM[lw.LevelM-1]))
		c := make(chan map[string]vclock.VClock, div)

		//divide up the creation of the lattice for a level
		for core := 0; core < div; core++ {
			go func(division int, comm chan map[string]vclock.VClock) {
				subMap := make(map[string]vclock.VClock, len(lw.LatticeM[lw.LevelM-1])*len(ids)/div)
				for j := len(lw.LatticeM[lw.LevelM-1]) / div * division; j < len(lw.LatticeM[lw.LevelM-1])/div*(division+1); j++ {
					p := lw.LatticeM[lw.LevelM-1][j]
					//fmt.Printf("Inital Point %s\n",p.ReturnVCString())
					for i, id := range ids {
						pu := p.Copy()
						pu.Tick(ids[i])
						vstring := pu.ReturnVCString()
						_, ok := subMap[vstring]
						if !ok && fastCorrectLatticePoint(mClocks, pu, id) {
							//fmt.Println(vstring)
							subMap[vstring] = pu
							levelPoints++
						}
					}
				}
				comm <- subMap
			}(core, c)
		}
		for i := 0; i < div; i++ {
			sm := <-c // wait for everyone to finnish
			for k, v := range sm {
				nextMap[k] = v
			}
			//fmt.Printf("Thread %d complete\n",i)
		}
		fmt.Printf("\rComputing lattice  %3.0f%% \t points %d\t fanout %d Threads %d", 100*float32(lw.LevelD)/float32(levels), lw.Points, len(lw.LatticeM[lw.LevelM-1]), div)
		lw.Push(mapToArray(nextMap))
		//levelToString(&lattice[lw.Level])

	}
	fmt.Println()
	return lw
}

//BuildLattice constructs a lattice based on an ordered set of vector clocks. The
//computed lattice is represented as a 2-D array of vector clocks
//[i][j] where the i is the level of the lattice, and j is a set of
//events at that level
//Build lattice 4 is threaded
func BuildLattice4(clocks [][]vclock.VClock) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	mClocks := clocksToMaps(clocks) //clockWrapper
	levels := sumTime(clocks)

	var numCPU = runtime.NumCPU()
	// add a common starting point to the lattice:
	// for every host (the first level of the clocks array) a new clock tick at time 0 is inserted
	for i := range clocks {
		latticePoint.Tick(ids[i])
	}

	// 'points' and 'levelPoints' are only needed for progress output, not for the algorithm itself
	level, points := 0, 0
	// make the total lattice the number of levels squared for safty equal to the size of the ids
	lattice := make([][]vclock.VClock, levels*levels)
	lattice[level] = make([]vclock.VClock, 1)
	lattice[level][0] = latticePoint

	// iterate over the levels, starting with level 1 as long as the lattice still has nodes on the next level
	for len(lattice[level]) > 0 {
		level++
		// next lattice level = previous *num nodes (worst case)
		nextMap := make(map[string]vclock.VClock, len(lattice[level-1])*len(ids))
		levelPoints := 0

		var div int
		//dont bother splitting if it's not worth it
		if len(lattice[level-1]) < 1000 {
			div = 1
		} else {
			div = numCPU
		}
		c := make(chan map[string]vclock.VClock, div)

		for core := 0; core < div; core++ {
			go func(division int, comm chan map[string]vclock.VClock) {
				subMap := make(map[string]vclock.VClock, len(lattice[level-1])*len(ids)/div)
				for j := len(lattice[level-1]) / div * division; j < len(lattice[level-1])/div*(division+1); j++ {
					p := lattice[level-1][j]
					//fmt.Printf("Inital Point %s\n",p.ReturnVCString())
					for i, id := range ids {
						pu := p.Copy()
						pu.Tick(ids[i])
						vstring := pu.ReturnVCString()
						_, ok := subMap[vstring]
						if !ok && fastCorrectLatticePoint(mClocks, pu, id) {
							//fmt.Println(vstring)
							subMap[vstring] = pu
							levelPoints++
						}
					}
				}
				comm <- subMap
			}(core, c)
		}
		for i := 0; i < div; i++ {
			sm := <-c // wait for everyone to finnish
			for k, v := range sm {
				nextMap[k] = v
			}
			//fmt.Printf("Thread %d complete\n",i)
		}
		lattice[level] = mapToArray(nextMap)
		fmt.Printf("\rComputing lattice  %3.0f%% \t points %d\t fanout %d Threads %d", 100*float32(level)/float32(levels), points, len(lattice[level]), div)
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
				if a[i][j].Compare(b[i][k], vclock.Equal) {
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

func mapToArray(vmap map[string]vclock.VClock) []vclock.VClock {
	array := make([]vclock.VClock, len(vmap))
	i := 0
	for _, clock := range vmap {
		array[i] = clock
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
		latticePoint.Tick(ids[i])
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
				pu.Tick(ids[i])
				if !queueContainsClock(next, pu) && correctLatticePoint(clocks[i], pu, ids[i]) {
					next.Add(pu)
					points++
				}
			}
			fmt.Printf("\rComputing lattice  %3.0f%% \t point %d\t fanout %d", 100*float32(level)/float32(levels), points, len(lattice[level]))
			lattice[len(lattice)-1] = append(lattice[len(lattice)-1], p.Copy())
		}
		level++
	}
	fmt.Println()
	return lattice
}

//queueContainsClock searches a queue of unorded vector clocks q for a
//match to the argument v. If v is in the queue return true, otherwise
//false
func queueContainsClock(q *queue.Queue, v vclock.VClock) bool {
	for i := 0; i < q.Length(); i++ {
		check := q.Get(i).(vclock.VClock)
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
func correctLatticePoint(loggedClocks []vclock.VClock, latticePoint vclock.VClock, id string) bool {
	found, index := searchClockById(loggedClocks, latticePoint, id)
	//if the exact value was not found, then it was a non logged local
	//event, in this case the vector clock previous to the recieve is
	//used
	if !found {
		index, found = nearestPrecedingClock(loggedClocks, latticePoint, index, id)
	}
	hostClock := loggedClocks[index]
	if LatticeValidWithLog(latticePoint, hostClock) && found {
		return true
	}
	return false
}

func fastCorrectLatticePoint(mClocks map[string]map[uint64]map[string]uint64, point vclock.VClock, id string) bool {
	clock, found := fastSearchClockById(mClocks, point, id)
	if !found {
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

func LatticeValidWithLog(latticePoint, loggedClock vclock.VClock) bool {
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
func nearestPrecedingClock(clocks []vclock.VClock, proposedClock vclock.VClock, index int, id string) (int, bool) {
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
