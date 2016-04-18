package ricartagrawala

import (
	"bitbucket.org/bestchai/dinv/instrumenter"
	"fmt"
	"math/rand"
	"net"
	"time"
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
	Id 		int
	Criticals int
}

type Report struct {
	Starved      bool
	Crashed      bool
	ErrorMessage error
	OtherDied    bool
	Criticals int
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

func critical() {
	instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
	report.Criticals++
	fmt.Printf("Running Critical Section on %d with Request time:%d \n", id, RequestTime)
}

func Host(idArg, hostsArg int, planArg Plan) Report {
	instrumenter.Initalize(fmt.Sprintf("%d-%d",idArg,planArg.Id))
	id = idArg
	hosts = hostsArg
	plan = planArg
	fmt.Printf("Starting %d\n", id+BASEPORT)
	initConnections(id, hosts)
	fmt.Printf("Connected to %d hosts on %d\n", len(nodes), id)
	time.Sleep(1000 * time.Millisecond)

	//start the receving demon
	go receive()
	//broadcast hello to everyone
	broadcast(fmt.Sprintf("Hello from %d", id))

	//local variables to track outstanding critical requests
	crit := false                 //true if critical section is requested
	outstanding := make([]int, 0) //messages being witheld
	okays := make(map[int]bool)
	done := make(map[int]bool,0)
	sentTime := time.Now()
	starving := make(map[int]bool)
	for true {

		if len(done) == hosts {
			fmt.Printf("Host %d done\n",plan.Id)
			break
		}

		if (plan.Criticals == report.Criticals){
			//everything is done
			done[plan.Id] = true
			broadcast("done")
		}

		if !crit {
			crit = (rand.Float64() > .99)
			if crit {
				RequestTime = Lamport
				outstanding = make([]int, 0)
				okays = make(map[int]bool)
				sentTime = time.Now()
				starving = make(map[int]bool)
				//fmt.Printf("Requesting Critical on %d at time %d\n",id,RequestTime)
				broadcast("critical")
				instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
			}
		}

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
				//fmt.Println(trues)

				break
			case "critical":
				if !crit || RequestTime > m.Lamport || (RequestTime == m.Lamport && id < m.Sender) {
					if crit {
						instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
						starving[m.Sender] = true
					}
					//fmt.Printf("sending ok to %d from %d Their Request Time:%d My Request Time: %d\n",m.Sender,id,m.Lamport,RequestTime)
					send("ok", m.Sender)
				} else {
					//fmt.Printf("withholding ok (%d/%d) on %d from request id:%d lamport:%d Requesting Time :%d\n",len(outstanding),hosts-1,id,m.Sender,m.Lamport,RequestTime)
					instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
					outstanding = append(outstanding, m.Sender)
				}
				break
			case "death":
				instrumenter.Dump("nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report",nodes,listen,Lamport,RequestTime,id,hosts,lastMessage,updated,plan,report)
				crashGracefully(fmt.Errorf("I %d Got the death message from %d now I die too\n", id, m.Sender))
				break
			case "done":
				done[m.Sender] = true
				break
			default:
				continue
			}
		}

		//timeout
		if crit && sentTime.Add(time.Millisecond*100).Before(time.Now()) {
			sentTime = time.Now()
			//fmt.Printf("Timeout ok %d",id)
			for i := 0; i < hosts; i++ {
				//dont make a connection with yourself
				if i == id || okays[i] {
					continue
				} else {
					send("critical", i)
				}
			}
		}

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
	fmt.Printf("exiting")
	return report
}

func receive() {
	//defer listen.Close()
	for true {
		buf := make([]byte, 1024)
		m := new(Message)
		//listen.SetDeadline(time.Now().Add(time.Second))
		n, err := listen.Read(buf[0:])
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
}

func broadcast(msg string) {
	//fmt.Printf("broadcasting %s\n",msg)
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

	out := instrumenter.Pack(m)
	//fmt.Printf("sending %s [ %d --> %d ] {body : %s}\n",msg,id, host,m.String())
	listen.WriteToUDP(out, nodes[host])
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
			crashGracefully(err)
		}
	}
}

func crashGracefully(err error) {
	if err != nil {
		fmt.Println(err)
		report.ErrorMessage = err
		report.Crashed = true
		fmt.Printf("broadcasting death on %d\n", id)
		broadcast("death")
	}
}
