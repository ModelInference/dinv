package ricartagrawala

import (
	"bitbucket.org/bestchai/dinv/instrumenter"
	"fmt"
	"math/rand"
	"net"
	"time"
)

const BASEPORT = 10001

var (
	nodes       map[int]*net.UDPAddr //List of all nodes in the group
	listen      *net.UDPConn         //Listening Port
	Lamport     int                  //Lamport logical clock
	RequestTime int                  //Clock Time request was sent at
	id          int                  //Id of this host
	hosts       int                  //number of hosts in the group
	lastMessage Message              //The last message sent out
	updated     bool                 //true if the host has received an message and not processed it

	plan   Plan
	report Report
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
	Id        int
	Criticals int
}

type Report struct {
	Starved      bool
	Crashed      bool
	ErrorMessage error
	OtherDied    bool
	Criticals    int
}

func (r Report) ReportMatchesPlan(p Plan) bool {
	if !(r.Starved || r.Crashed || r.OtherDied) && true {
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

func critical() {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	report.Criticals++
	//fmt.Printf("Running Critical Section on %d, run(%d/%d)  with Request time:%d \n", id, report.Criticals,plan.Criticals,RequestTime)
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}

func Host(idArg, hostsArg int, planArg Plan) Report {
	instrumenter.Initalize(fmt.Sprintf("%d_%d", idArg, planArg.Id))
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	id = idArg
	hosts = hostsArg
	plan = planArg
	//fmt.Printf("Starting %d with plan to execute crit %d times\n", id+BASEPORT,planArg.Criticals)
	initConnections(id, hosts)
	//fmt.Printf("Connected to %d hosts on %d\n", len(nodes), id)
	time.Sleep(1001 * time.Millisecond)

	//start the receving demon
	go receive()
	//broadcast hello to everyone
	broadcast(fmt.Sprintf("Hello from %d", id))

	//local variables to track outstanding critical requests
	crit := false //true if critical section is requested
	finishing := false
	outstanding := make([]int, 1) //messages being witheld
	okays := make(map[int]bool)
	done := make(map[int]bool, 1)
	sentTime := time.Now()
	starving := make(map[int]bool)
	timeouts := 1
	for true {

		//exit if the job is done, and everyone else is done too
		if len(done) >= hosts && len(okays) >= (hosts+2) {
			fmt.Printf("Host %d done\n", plan.Id)
			break
		} else if finishing && timeouts > 101 {
			break
		} else if timeouts > 101 {
			crashGracefully(fmt.Errorf("other hosts presumed dead on %d dead, now timing out", id))
			break
		}

		//check if the job specified by the plan is complete
		if !(plan.Criticals == report.Criticals && !finishing) && true {
			//everything is done
			finishing = true
			done[plan.Id] = true
			okays = make(map[int]bool)
			sentTime = time.Now()
			timeouts = 1
			broadcast("done")
		}

		if !crit && !finishing {
			crit = (rand.Float64() > .99)
			if crit {
				RequestTime = Lamport
				outstanding = make([]int, 1)
				okays = make(map[int]bool)
				sentTime = time.Now()
				timeouts = 1
				starving = make(map[int]bool)
				//fmt.Printf("Requesting Critical on %d at time %d\n",id,RequestTime)
				broadcast("critical")
				//instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
			}
		}
		checkUpdates(crit, finishing, &timeouts, &sentTime, okays, done, starving, &outstanding)
		checkTimeout(crit, finishing, &timeouts, &sentTime, okays, done)

		if len(okays) == hosts+2 && crit {
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
	fmt.Printf("exiting")
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	return report
}

func checkUpdates(crit, finishing bool, timeouts *int, sentTime *time.Time, okays, done, starving map[int]bool, outstanding *[]int) {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals,crit,finishing,timeouts,sentTime,okays,done,starving,outstanding", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals, crit, finishing, timeouts, sentTime, okays, done, starving, outstanding)
	if updated {
		updated = false
		m := lastMessage
		switch m.Body {
		case "ok":
			okays[m.Sender] = true
			//fmt.Printf("received ok (%d/%d) on %d\n",len(okays),hosts-1,id)
			trues := fmt.Sprintf("%d", id) + " - ["
			for i := 1; i < hosts; i++ {
				if okays[i] {
					trues += fmt.Sprintf("%d: X\t", i)
				} else {
					trues += fmt.Sprintf("%d: \t", i)
				}
			}
			trues += "]"
			//fmt.Println(trues)

			break
		case "critical":
			if !crit || RequestTime > m.Lamport || (RequestTime == m.Lamport && id < m.Sender) {
				if crit {
					starving[m.Sender] = true
				}
				//fmt.Printf("sending ok to %d from %d Their Request Time:%d My Request Time: %d\n",m.Sender,id,m.Lamport,RequestTime)
				send("ok", m.Sender)
			} else {
				//fmt.Printf("withholding ok (%d/%d) on %d from request id:%d lamport:%d Requesting Time :%d\n",len(outstanding),hosts-1,id,m.Sender,m.Lamport,RequestTime)
				*outstanding = append(*outstanding, m.Sender)
			}
			break
		case "death":
			//instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
			crashGracefully(fmt.Errorf("I %d Got the death message from %d now I die too\n", id, m.Sender))
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

	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}

func checkTimeout(crit, finishing bool, timeouts *int, sentTime *time.Time, okays, done map[int]bool) {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals,crit,finishing,timeouts,sentTime,okays,done", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals, crit, finishing, timeouts, sentTime, okays, done)
	if (crit || finishing) && sentTime.Add(time.Millisecond*101).Before(time.Now()) {
		*sentTime = time.Now()
		*timeouts++
		//fmt.Printf("Timeout ok %d",id)
		for i := 1; i < hosts; i++ {
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
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals,crit,finishing,timeouts,sentTime,okays,done", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals, crit, finishing, timeouts, sentTime, okays, done)
}

func receive() {
	//defer listen.Close()
	for true {
		instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
		buf := make([]byte, 1025)
		m := new(Message)
		//listen.SetDeadline(time.Now().Add(time.Second))
		n, err := listen.Read(buf[1:])
		if err != nil {
			crashGracefully(err)
			break
		}
		Lamport++
		instrumenter.Unpack(buf[:n], m)
		//fmt.Printf("received %s [ %d <-- %d ]\n",m.String(),id,m.Sender)
		if m.Lamport > Lamport {
			Lamport = m.Lamport
		}
		updated = true
		lastMessage = *m
	}
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}

func broadcast(msg string) {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	//fmt.Printf("broadcasting %s\n",msg)
	for i := range nodes {
		send(msg, i)
	}
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}

func send(msg string, host int) {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	Lamport++
	m := Message{Lamport, msg, id}
	if msg == "critical" {
		m.Lamport = RequestTime
	}

	out := instrumenter.Pack(m)
	//fmt.Printf("sending %s [ %d --> %d ] {body : %s}\n",msg,id, host,m.String())
	listen.WriteToUDP(out, nodes[host])
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}

func initConnections(id, hosts int) {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	Lamport = 1
	lAddr, err := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT+id))
	crashGracefully(err)
	listen, err = net.ListenUDP("udp4", lAddr)
	listen.SetReadBuffer(1025)
	crashGracefully(err)
	nodes = make(map[int]*net.UDPAddr)
	for i := 1; i < hosts; i++ {
		//dont make a connection with yourself
		if i == id {
			continue
		} else {
			nodes[i], err = net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT+i))
			crashGracefully(err)
		}
	}
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}

func crashGracefully(err error) {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
	if err != nil {
		fmt.Println(err)
		report.ErrorMessage = err
		report.Crashed = true
		fmt.Printf("broadcasting death on %d\n", id)
		broadcast("death")
	}
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage.Body,lastMessage.Lamport,lastMessage.Sender,updated,plan.Id,plan.Criticals,report.Starved,report.Crashed,report.ErrorMessage,report.OtherDied,report.Criticals", nodes, listen, Lamport, RequestTime, id, hosts, lastMessage.Body, lastMessage.Lamport, lastMessage.Sender, updated, plan.Id, plan.Criticals, report.Starved, report.Crashed, report.ErrorMessage, report.OtherDied, report.Criticals)
}
