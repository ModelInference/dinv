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

	"bitbucket.org/bestchai/dinv/govec/vclock"
	"gopkg.in/eapache/queue.v1"
)

//BuildLattice constructs a lattice based on an ordered set of vector clocks. The
//computed lattice is represented as a 2-D array of vector clocks
//[i][j] where the i is the level of the lattice, and j is a set of
//events at that level
func BuildLattice(clocks [][]vclock.VClock) [][]vclock.VClock {
	latticePoint := vclock.New()
	//initalize lattice clock
	ids := idClockMapper(clocks)
	for i := range clocks {
		latticePoint.Update(ids[i], 0)
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
