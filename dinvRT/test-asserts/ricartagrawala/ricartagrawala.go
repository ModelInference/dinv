package main

import (
	"bitbucket.org/bestchai/dinv/dinvRT"
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
	
        "github.com/acarb95/DistributedAsserts/assert"
)

const BASEPORT = 10000

var (
	nodes       map[int]*net.UDPAddr //List of all nodes in the group
	listen      *net.UDPConn         //Listening Port
	Lamport     int                  //Lamport logical clock
	RequestTime int                  //Clock Time request was sent at
	id          int                  //Id of this host
	hosts       int                  //number of hosts in the group
	lastMessage Message              //The last message sent out
	updated     bool                 //true if the host has received an message and not processed it
	startTime   time.Time            //The startup time of the node

	plan   Plan
	report Report

	inCritical bool
	logger     *log.Logger
)

type Message struct {
	Lamport int
	Body    string
	Sender  int
}

func (m *Message) String() string {
	return fmt.Sprintf("[ %s | %d | clock(%d)]", m.Body, m.Sender, m.Lamport)
}

type Plan struct {
	Id            int
	Criticals     int
	GlobalTimeout int
}

type Report struct {
	Starved      bool
	Crashed      bool
	ErrorMessage error
	OtherDied    bool
	Criticals    int
}

func (r Report) ReportMatchesPlan(p Plan) bool {
	if r.Starved || r.Crashed || r.OtherDied {
		return false
	}
	if r.ErrorMessage != nil {
		return false
	}
	if p.Criticals == r.Criticals {
		return true
	}
	return true
}

// ============================== ASSERT CODE ==============================
func assertValue(values map[string]map[string]interface{}) bool {
	crit := false
	for _, v := range values {
		// fmt.Printf("%s: %t\n", k, v["inCritical"].(bool))
		if v["inCritical"].(bool) {
			if (crit) {
				return false
			}
			crit = true
		}
	}
	return true
}
// ============================ END ASSERT CODE ============================

func critical() {
	inCritical = true
	dinvRT.Track(fmt.Sprintf("%dC", plan.Id), "crictical", inCritical)

	// ============================== ASSERT CODE ==============================
	requestedValues := make(map[string][]string);
	requestedValues[":"+fmt.Sprintf("%d", BASEPORT+id+2*hosts)] = append(requestedValues[":"+fmt.Sprintf("%d", BASEPORT+id+2*hosts)], "inCritical")
	for _, v := range neighbors {
	 	requestedValues[v] = append(requestedValues[v], "inCritical")
	}
	assert.Assert(assertValue, requestedValues)
	// ============================ END ASSERT CODE ============================
	
	report.Criticals++

	// ============================== ASSERT CODE ==============================
	inCritical = false // CHANGE TO: comment to induce assertion failure
	// ============================ END ASSERT CODE ============================
}

var (
	idInput    int
	hostsInput int
	timeInput  int
)

var neighbors = []string{}

func main() {
	// fmt.Println("STARTING")
	var idarg = flag.Int("id", 0, "hosts id")
	var hostsarg = flag.Int("hosts", 0, "#of hosts")
	var timearg = flag.Int("time", 0, "timeout")
	flag.Parse()
	idInput = *idarg
	hostsInput = *hostsarg
	timeInput = *timearg
	plan := Plan{idInput, 10, timeInput}
	// fmt.Println(plan.Criticals)
	report := Host(idInput, hostsInput, plan)
	if !report.ReportMatchesPlan(plan) {
		// fmt.Println("FAILED")

	} else {
		// fmt.Println("PASSED")
	}
}

func Host(idArg, hostsArg int, planArg Plan) Report {
	id = idArg
	hosts = hostsArg
	plan = planArg
	startTime = time.Now()
	// fmt.Printf("Starting %d with plan to execute crit %d times\n", id+BASEPORT, planArg.Criticals)
	initConnections(id, hosts)
	// fmt.Printf("%d: Connected to %d hosts on %d\n", id, len(nodes), id)
	time.Sleep(5 * time.Second)
	// fmt.Printf("%d Done waiting\n", id)

	//start the receving demon
	go receive()
	//broadcast hello to everyone
	broadcast(fmt.Sprintf("Hello from %d", id))

	//local variables to track outstanding critical requests
	crit := false //true if critical section is requested
	finishing := false
	outstanding := make([]int, 0) //messages being witheld
	okays := make(map[int]bool)
	done := make(map[int]bool, 0)
	sentTime := time.Now()
	starving := make(map[int]bool)
	timeouts := 0
	for true {

		//exit if the job is done, and everyone else is done too
		if len(done) >= hosts && len(okays) >= (hosts-1) {
			// fmt.Printf("Host %d done\n", plan.Id)
			break
		} else if finishing && timeouts > 10 {
			break
		} else if startTime.Add(time.Second * time.Duration(plan.GlobalTimeout)).Before(time.Now()) {
			// fmt.Printf("TIMEOUT\n")
			break
		}

		//check if the job specified by the plan is complete
		if plan.Criticals == report.Criticals && !finishing {
			//everything is done
			finishing = true
			done[plan.Id] = true
			okays = make(map[int]bool)
			sentTime = time.Now()
			timeouts = 0
			broadcast("done")
		}

		if !crit && !finishing {
			crit = (rand.Float64() > .8)
			if crit {
				RequestTime = Lamport
				outstanding = make([]int, 0)
				okays = make(map[int]bool)
				sentTime = time.Now()
				timeouts = 0
				starving = make(map[int]bool)
				//fmt.Printf("Requesting Critical on %d at time %d\n",id,RequestTime)
				broadcast("critical")
			}
		}
		checkUpdates(crit, finishing, &timeouts, &sentTime, okays, done, starving, &outstanding)
		checkTimeout(crit, finishing, &timeouts, &sentTime, okays, done)

		if len(okays) == hosts-1 && crit {
			critical()
			crit = false
			for _, n := range outstanding {
				send("ok", n)
			}
		}

		if len(starving) >= hosts {
			report.Starved = true
			crashGracefully(fmt.Errorf("Starved to death on host %d\n", id))
		}

		if report.Crashed {
			break
		}
	}
	// fmt.Printf("exiting")
	return report
}

func checkUpdates(crit, finishing bool, timeouts *int, sentTime *time.Time, okays, done, starving map[int]bool, outstanding *[]int) {
	if updated {
		updated = false
		m := lastMessage
		switch m.Body {
		case "ok":
			okays[m.Sender] = true
			//fmt.Printf("received ok (%d/%d) on %d\n",len(okays),hosts-1,id)
			trues := fmt.Sprintf("%d", id) + " - ["
			for i := 0; i < hosts; i++ {
				if okays[i] {
					trues += fmt.Sprintf("%d: X\t", i)
				} else {
					trues += fmt.Sprintf("%d: \t", i)
				}
			}
			trues += "]"
			// fmt.Println(trues)

			break
		case "critical":
			if !crit || RequestTime > m.Lamport || (RequestTime == m.Lamport && id < m.Sender) {
				if crit {
					starving[m.Sender] = true
				}
				//fmt.Printf("sending ok to %d from %d Their Request Time:%d My Request Time: %d\n",m.Sender,id,m.Lamport,RequestTime)
				send("ok", m.Sender)
			} else {
				//fmt.Printf("withholding ok (%d/%d) on %d from request id:%d lamport:%d Requesting Time :%d\n",len(*outstanding),hosts-1,id,m.Sender,m.Lamport,RequestTime)
				*outstanding = append(*outstanding, m.Sender)
			}
			break
		case "death":
			//crashGracefully(fmt.Errorf("I %d Got the death message from %d now I die too\n", id, m.Sender))
			break
		case "done":
			done[m.Sender] = true
			//fmt.Printf("sending done ok to %d from %d Their Request Time:%d My Request Time: %d recieved %d/%d \n",m.Sender,id,m.Lamport,RequestTime,len(done),hosts-1)
			send("ok", m.Sender)
			break
		default:
			//break TODO check if this breaks anything
			return
		}
	}

}

func checkTimeout(crit, finishing bool, timeouts *int, sentTime *time.Time, okays, done map[int]bool) {
	if (crit || finishing) && sentTime.Add(time.Millisecond*100).Before(time.Now()) {
		*sentTime = time.Now()
		*timeouts++
		//fmt.Printf("Timeout ok %d",id)
		for i := 0; i < hosts; i++ {
			//dont make a connection with yourself
			if i == id || okays[i] {
				continue
			} else {
				if crit {
					send("critical", i)
				} else {
					//fmt.Printf("resending done %d/%d criticals %d/%d dones",report.Criticals,plan.Criticals,len(done),hosts)
					send("done", i)
				}
			}
		}
	}
}

func receive() {
	//defer listen.Close()
	for true {
		buf := make([]byte, 1024)
		m := new(Message)
		//listen.SetDeadline(time.Now().Add(time.Second))
		n, err := capture.Read(listen.Read, buf)
		//fmt.Printf("Received :%s\n%d\n",buf,n)
		if err != nil {
			crashGracefully(err)
			//break
		}
		network := bytes.NewReader(buf[:n])
		dec := gob.NewDecoder(network)
		err = dec.Decode(&m)
		if err != nil {
			crashGracefully(err)
			break
		}
		Lamport++

		//dinvRT.Unpack(buf[:n], m)
		//fmt.Printf("received %s [ %d --> %d ]\n",m.String(),m.Sender,id)
		if m.Lamport > Lamport {
			Lamport = m.Lamport
		}
		updated = true
		lastMessage = *m
	}
}

func broadcast(msg string) {
	// fmt.Printf("broadcasting %s\n", msg)
	for i := range nodes {
		send(msg, i)
	}
}

func send(msg string, host int) {
	Lamport++
	m := Message{Lamport, msg, id}
	if msg == "critical" {
		m.Lamport = RequestTime
	}
	//endcode
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(m)
	if err != nil {
		log.Fatal("encode :", err)
	}
	//fmt.Println(network.Bytes())
	_, err = capture.WriteToUDP(listen.WriteToUDP, network.Bytes(), nodes[host])
	dinvRT.Track(fmt.Sprintf("%dS", plan.Id), "crictical", inCritical)
	if err != nil {
		crashGracefully(err)
	}

}

func initConnections(id, hosts int) {
	Lamport = 0
	lAddr, err := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT+id))
	crashGracefully(err)
	listen, err = net.ListenUDP("udp4", lAddr)
	listen.SetReadBuffer(1024)
	crashGracefully(err)
	nodes = make(map[int]*net.UDPAddr)
	for i := 0; i < hosts; i++ {
		//dont make a connection with yourself
		if i == id {
			continue
		} else {
			nodes[i], err = net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT+i))
			neighbors = append(neighbors,":"+fmt.Sprintf("%d", BASEPORT+i+2*hosts))
			// fmt.Println(nodes[i])
			crashGracefully(err)
		}
	}

	// ============================== ASSERT CODE ==============================
	processName := fmt.Sprintf("node%d", id)
	assert.InitDistributedAssert(":"+fmt.Sprintf("%d", BASEPORT+id+2*hosts), neighbors, processName);
	assert.AddAssertable("inCritical", &inCritical, nil);
	// ============================ END ASSERT CODE ============================
}

func crashGracefully(err error) {
	if err != nil {
		if err == io.EOF {
			fmt.Println("dont worry about eof")
			return
		}
		fmt.Println(err)
		report.ErrorMessage = err
		report.Crashed = true
		// fmt.Printf("broadcasting death on %d\n", id)
		broadcast("death")
	}
}
