/*
A cut of a trace of vector clocks can be represented by a set of clock
values, one per host. Cuts can have a number of properties, such as
consistency, and subsets of clocks which can be totally orderd. Here
these features of cuts are calculated based on logs of vector clocks,
and lattices constructed from those logs.

Author: Stewart Grant
Edited: July 6 2015
*/

package logmerger

import (
	"fmt"

	"github.com/arcaneiceman/GoVector/govec/vclock"
)

//mineConsistentCuts determines every consistent cut occuring in a log
//of vector clocks. lattice is the set of all potential event
//orderings in the log. clocks is the logs of vector clocks. deltaComm
//is an enumerated history of all sends and recieves on each host in
//clocks.
func mineConsistentCuts(lattice [][]vclock.VClock, clocks [][]vclock.VClock, deltaComm [][]int) []Cut {
	ids := idClockMapper(clocks)
	consistentCuts := make([]Cut, 0)
	for i := range lattice {
		for j := range lattice[i] {
			communicationDelta := 0
			// TODO preallocate by some heuristic?
			var potentialCut Cut
			for k := range ids {
				ticks, found := lattice[i][j].FindTicks(ids[k])
				if !found {
					break
				}
				found, index := searchClockById(clocks[k], lattice[i][j], ids[k])
				if !found {
					fmt.Printf("\rCant Find matching clock %s - %d -> %s \t insted found %s", ids[k], ticks, lattice[i][j].ReturnVCString(), clocks[k][index].ReturnVCString())
					break
				}
				//fmt.Printf("%d", communicationDelta)
				communicationDelta += deltaComm[k][index]
				potentialCut.Clocks = append(potentialCut.Clocks, clocks[k][index])
			}
			if communicationDelta == 0 {
				fmt.Printf("\rcomputing cuts %3.0f%%  \t[%d] found", 100*float32(i)/float32(len(lattice)), len(consistentCuts))
				//logger.Printf("%s\n", potentialCut.String())
				consistentCuts = append(consistentCuts, potentialCut)
			}
		}
	}
	fmt.Println()
	return consistentCuts
}

//mineConsistentCuts determines every consistent cut occuring in a log
//of vector clocks. lattice is the set of all potential event
//orderings in the log. clocks is the logs of vector clocks. deltaComm
//is an enumerated history of all sends and recieves on each host in
//clocks.
func mineConsistentCuts2(lw *LatticeWrapper, clocks [][]vclock.VClock, deltaComm [][]int) []Cut {
	ids := idClockMapper(clocks)
	consistentCuts := make([]Cut, 0)
	lw.Beginning()
	defer lw.Delete()

	for level := lw.Pop(); level != nil; level = lw.Pop() {

		div := ThreadCount(len(lw.LatticeM[lw.LevelM-1]))
		c := make(chan []Cut, div)

		//divide up the creation of the lattice for a level
		for core := 0; core < div; core++ {
			go func(division int, comm chan []Cut) {
				cuts := make([]Cut, len(lw.LatticeM[lw.LevelM-1])/div)
				cutsFound := 0
				for j := len(level) / div * division; j < len(level)/div*(division+1); j++ {
					//fmt.Println(j)
					communicationDelta := 0
					// TODO preallocate by some heuristic?
					var potentialCut Cut
					for k := range ids {
						ticks, found := level[j].FindTicks(ids[k])
						if !found {
							break
						}
						found, index := searchClockById(clocks[k], level[j], ids[k])
						if !found {
							fmt.Printf("\rCant Find matching clock %s - %d -> %s \t insted found %s", ids[k], ticks, level[j].ReturnVCString(), clocks[k][index].ReturnVCString())
							break
						}
						//fmt.Printf("%d", communicationDelta)
						communicationDelta += deltaComm[k][index]
						potentialCut.Clocks = append(potentialCut.Clocks, clocks[k][index])
					}
					if communicationDelta == 0 {
						cuts[cutsFound] = potentialCut
						cutsFound++
					}
				}
				comm <- cuts[0:cutsFound]
			}(core, c)
		}
		for i := 0; i < div; i++ {
			cuts := <-c // wait for everyone to finnish
			consistentCuts = append(consistentCuts, cuts...)
			//fmt.Printf("Thread %d complete\n",i)
		}
		fmt.Printf("\rcomputing cuts %3.0f%%  \t[%d] found Threads %d", 100*float32(lw.LevelM)/float32(len(level)), len(consistentCuts), div)
		//logger.Printf("%s\n", potentialCut.String())
	}
	fmt.Println()
	return consistentCuts
}

//within a cut subsets of clocks can be totally ordered with one
//another. These orderings are extracted from the log of clocks, are
//retured as sets of indexed clock values. Where -> denotes a send and
//matching receieve. Example: i -> j -> k, and x -> y, the returned
//indexes would be [[k,j,i],[y,x]]
func totalOrderFromCut(cut Cut, clocks [][]vclock.VClock) [][]int {
	allreadyOrdered := make([]bool, len(cut.Clocks))
	ancestors := countAncestors(cut)
	ids := idClockMapper(clocks)
	ordering := make([][]int, 0)
	for true {
		//find the root host of the totally ordered chain
		rootHost, found := findUnorderedHost(ancestors, allreadyOrdered)
		if !found {
			break
		}
		ordering = append(ordering, make([]int, 0))
		appendOrdering(ordering, allreadyOrdered, rootHost)
		//find all other hosts in the totally ordered chain
		for true {
			sendersIndex, found := findSenderInCut(cut, clocks, allreadyOrdered, ids, rootHost)
			if !found {
				break
			}
			appendOrdering(ordering, allreadyOrdered, sendersIndex)
			rootHost = sendersIndex
		}
	}
	return ordering
}

func appendOrdering(ordering [][]int, ordered []bool, hostIndex int) {
	ordering[len(ordering)-1] = append(ordering[len(ordering)-1], hostIndex)
	ordered[hostIndex] = true
}

func findSenderInCut(cut Cut, clocks [][]vclock.VClock, ordered []bool, ids []string, receiver int) (sender int, found bool) {
	sender, found = -1, false
	newestEvent := -1
	//find the most recent receive on host rootHost
	for potentialSender := range cut.Clocks {
		if potentialSender != receiver && !ordered[potentialSender] {
			matchedReceiver, event, matched := matchSendAndReceive(cut.Clocks[potentialSender], clocks, ids[potentialSender])
			if matched && matchedReceiver == receiver && event > newestEvent {
				newestEvent, sender, found = event, potentialSender, true
			}
		}
	}
	return sender, found
}

func findUnorderedHost(ancestors []int, ordered []bool) (unorderedHost int, found bool) {
	//get oldest clock
	unorderedHost, found = -1, false
	maxAncestors := -1
	for i := range ancestors {
		//search for a clock that has yet to be allreadyOrdered with the
		//newest value
		if ancestors[i] > maxAncestors && !ordered[i] {
			maxAncestors, unorderedHost, found = ancestors[i], i, true
		}
	}
	return unorderedHost, found
}

//countAncestors returns the number of ancestors each clock in a cut
//has within the same cut. The number of ancestors is returned as an
//array of ints, where the index of the array corresponds to the clock
//index in the cut.
func countAncestors(cut Cut) []int {
	ancestors := make([]int, len(cut.Clocks))
	for i := range cut.Clocks {
		for j := range cut.Clocks {
			if i != j && cut.Clocks[i].Compare(cut.Clocks[j], vclock.Ancestor) {
				ancestors[i]++
			}
		}
	}
	return ancestors
}

//fillCommunicationDelta markes the difference in sends and recieves that
//have occured on a particualr host at every point throughout there
//exectuion
// a host with 5 sends and 2 recieves will be given the delta = 3
// a host with 10 receives and 5 sends will be given the delta = -5
func fillCommunicationDelta(commDelta [][]int) [][]int {
	for i := range commDelta {
		fill := 0
		for j := range commDelta[i] {
			if commDelta[i][j] != 0 {
				temp := commDelta[i][j]
				commDelta[i][j] += fill
				fill += temp
			} else {
				commDelta[i][j] += fill
			}
		}
	}
	return commDelta
}

//enumerateCommunication searches through a log of vector clocks
//matching sends and receives, the number of sends and recives are
//tallyed at each program point and the enumerated values are returned
func enumerateCommunication(clocks [][]vclock.VClock) [][]int {
	ids := idClockMapper(clocks)
	commDelta := make([][]int, len(clocks))
	for i := range clocks {
		commDelta[i] = make([]int, len(clocks[i]))
	}
	for i := range clocks {
		var lastSend vclock.VClock
		for j := range clocks[i] {
			receiver, receiverEvent, matched := matchSendAndReceive(clocks[i][j], clocks, ids[i])
			if matched {
				if lastSend != nil && clocks[i][j].Compare(lastSend, vclock.Equal) { //dont enumerate if the time has not changed since the last send
					//fmt.Printf("Ignoring duplicate clocks %s <--> %s\n", clocks[i][j].ReturnVCString(), lastSend.ReturnVCString())
					continue
				}
				commDelta[i][j]++
				commDelta[receiver][receiverEvent]--
				logger.Printf("SR pair found %s, %s\n", clocks[i][j].ReturnVCString(), clocks[receiver][receiverEvent].ReturnVCString())
				logger.Printf("Sender %s:%d ----> Receiver %s:%d\n", ids[i], commDelta[i][j], ids[receiver], commDelta[receiver][receiverEvent])
				lastSend = clocks[i][j].Copy()
			}
		}
	}
	commDelta = fillCommunicationDelta(commDelta)
	//fill in the blanks
	return commDelta
}

//Cut is an array of vector clocks, with a one to one relationship
//between clocks and hosts.
type Cut struct {
	Clocks []vclock.VClock
}

//returns true if the calling cut happend before the argument
func (c Cut) HappenedBefore(other Cut) bool {
	for _, beforeClock := range c.Clocks {
		for _, afterClock := range other.Clocks {
			//print(afterClock.ReturnVCString, beforeClock.ReturnVCString)
			if !beforeClock.Compare(afterClock, vclock.Descendant) {
				return false
			}
		}
	}
	return true
}

//Retruns a String representation of a cut. The string contains a list
//of each vector clock, on each host in the cut.
func (c Cut) String() string {
	catString := fmt.Sprintf("{")
	for _, clock := range c.Clocks {
		catString = fmt.Sprintf("%s |(VC: %s)", catString, clock.ReturnVCString())
	}
	catString = fmt.Sprintf("%s}", catString)
	return catString
}
