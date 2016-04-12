package main

import (
	"os"
	"strconv"
	"net"
	"fmt"
	"time"
	"math/rand"
	"bitbucket.org/bestchai/dinv/instrumenter"
)

const BASEPORT  = 10000

var (
	nodes map[int]*net.UDPAddr
	listen *net.UDPConn
	Lamport int
	id int
	hosts int
	lastMessage Message
	updated bool
)

type Message struct {
	Lamport int
	Body string
	Sender int
}

func (m *Message) String () string{
	return fmt.Sprintf("[ %s | %d | clock(%d)]",m.Body,m.Sender,m.Lamport)
}

func critical() {
	fmt.Printf("You might be the sickest node of them all : %d\n",id)
}

func main() {
	id, _ = strconv.Atoi(os.Args[1])
	hosts, _ = strconv.Atoi(os.Args[2])
	fmt.Printf("Starting %d\n",id+BASEPORT)
	//for len(nodes) < hosts -1 {
		initConnections(id,hosts)
		fmt.Printf("Connected to %d hosts on %d\n",len(nodes),id)
		time.Sleep(1000 * time.Millisecond)
	//}

	//start the receving demon
	go receive()
	broadcast(fmt.Sprintf("Hello from %d",id))

	crit := false
	outstanding := make([]int,0)
	okays := make(map[int]bool)
	sentTime := time.Now()
	for true {
		if !crit {
			crit = (rand.Float64() > .99)
			if crit {
				fmt.Printf("Requesting Critical on %d\n",id)
				broadcast("critical")
				outstanding = make([]int,0)
				okays = make(map[int]bool)
				sentTime = time.Now()
			}
		}
		
		//send("ok",(id + 1)%hosts)

		if updated {
			updated = false
			m := lastMessage
			switch m.Body {
			case "ok":
				okays[m.Sender]=true
				fmt.Printf("received ok (%d/%d) on %d\n",len(okays),hosts-1,id)
				trues := "["
				for i:= 0; i< hosts; i++ {
					if okays[i] {
						trues += fmt.Sprintf("%d: X\t",i)
					} else {
						trues += fmt.Sprintf("%d: \t",i)
					}
				}
				trues += "]"
				fmt.Println(trues)

				break
			case "critical":
				if Lamport > m.Lamport || (Lamport == m.Lamport && id < m.Sender) || !crit {
					send("ok",m.Sender)
				} else {
					fmt.Printf("withholding ok (%d/%d) on %d\n",len(outstanding),hosts-1,id)
					outstanding = append(outstanding,m.Sender)
				}
				break
			default:
				continue
			}
		}
	
		//timeout
		if crit && sentTime.Add(time.Second * 2).Before(time.Now()){
			sentTime = time.Now()
			//fmt.Printf("Timeout ok %d",id)
			for i:=0; i<hosts; i++{
				//dont make a connection with yourself
				if i == id || okays[i] {
					continue
				} else {
					send("critical",i)
				}
			}
		}

		if len(okays) == hosts -1 && crit {
			critical()
			crit = false
			for _, n := range outstanding {
				send("ok",n)
			}
		}


	}
}


func receive() {
	for true {
		buf := make([]byte,1024)
		m := new(Message)
		listen.SetDeadline(time.Now().Add(time.Second))
		n, err := listen.Read(buf[0:])
		if err != nil {
			//printErr(err)
			continue
		} 
		Lamport++
		instrumenter.Unpack(buf[:n],m)
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
		send(msg,i)
	}
}

func send(msg string, host int) {
		Lamport++
		m := Message{Lamport,msg,id}

		out := instrumenter.Pack(m)
		//fmt.Printf("sending %s [ %d --> %d ] {body : %s}\n",msg,id, host,m.String())
		listen.WriteToUDP(out,nodes[host])
		time.Sleep(1000 * time.Millisecond)
}


func initConnections(id, hosts int){
	Lamport = 0
	lAddr, err := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT + id))
	printErr(err)
	listen, err = net.ListenUDP("udp4",lAddr)
	listen.SetReadBuffer(1024)
	printErr(err)
	nodes = make(map[int]*net.UDPAddr)
	for i:=0; i<hosts; i++{
		//dont make a connection with yourself
		if i == id {
			continue
		} else {
			nodes[i], err = net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT + i))
			printErr(err)
			//nodes[BASEPORT + i], err = net.DialUDP("udp",nil,addr)
			//printErr(err)
		}
	}
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
