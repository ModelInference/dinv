package main

import (
	"bitbucket.org/bestchai/dinv/dinvRT"
	"flag"
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"math/rand"
	"net"
	"os"
	"time"
)

const (
	Ack          = 0xFF
	RequestStick = 1
	ReleaseStick = 2
	ExcuseMe     = 3
	SIZEOFINT    = 4
	n            = 50
	BUFF_SIZE    = 1024
	SLEEP_MAX    = 1e8
)

//global state variables
var (
	Eating         bool
	Thinking       bool
	LeftChopstick  bool
	RightChopstick bool
	Excused        bool
	ID             string
)

//Transition into the eating state
func EatingState() {
	Eating = true
	Thinking = false
	LeftChopstick = true
	RightChopstick = true
}

//transition into the thinking state
func ThinkingState() {
	Eating = false
	Thinking = true
	LeftChopstick = false
	RightChopstick = false
}

//obtain the left chopstick
func LeftChopstickState() {
	Eating = false
	Thinking = true
	LeftChopstick = true
}

//obtain the right chopstick
func RightChopstickState() {
	Eating = false
	Thinking = true
	RightChopstick = true
}

//structure defining a philosopher
type Philosopher struct {
	id, neighbourId int
	chopstick       chan bool // the left chopstick // inspired by the wikipedia page, left chopsticks should be used first
	neighbour       *net.UDPConn
}

func makePhilosopher(port, neighbourPort int) *Philosopher {
	fmt.Printf("Setting up listing connection on %d\n", port)
	conn, err := net.ListenPacket("udp", ":"+fmt.Sprintf("%d", port))
	if err != nil {
		panic(err)
	}

	//for general testing
	if port == 4000 {
		ID = fmt.Sprintf("%d", port)
	} else {
		ID = "ALL"
	}

	fmt.Printf("listening on %d\n", port)
	var neighbour *net.UDPConn
	var errDial error
	connected := false
	//Continuously try to connect via udp
	for !connected {
		neighbourAddr, errR := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", neighbourPort))
		PrintErr(errR)
		listenAddr, errL := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", port+1000))
		PrintErr(errL)
		neighbour, errDial = net.DialUDP("udp", listenAddr, neighbourAddr)
		PrintErr(errDial)
		connected = (errR == nil) && (errL == nil) && (errDial == nil)
	}

	//setup chopstick channel triggered when chopsticks are given or
	//received
	chopstick := make(chan bool, 1)
	chopstick <- true
	fmt.Printf("launching chopstick server\n")
	go func() {
		defer fmt.Printf("Chopstick #%d\n is down", port) //attempt to show when the chopsticks are no longer available
		//Incomming request handler
		for true {
			dinvRT.Track(fmt.Sprintf("%d", ID), "Excused,Ack,ReleaseStick,ExcuseMe,SLEEP_MAX,Eating,RightChopstick,n,BUFF_SIZE,Thinking,RequestStick,SIZEOFINT,LeftChopstick", Excused, Ack, ReleaseStick, ExcuseMe, SLEEP_MAX, Eating, RightChopstick, n, BUFF_SIZE, Thinking, RequestStick, SIZEOFINT, LeftChopstick)
			req, addr := getRequest(conn)
			go func(request int) {
				switch request {
				case ReleaseStick:
					fmt.Printf("Receiving stick on %d\n", port)
					chopstick <- true
				case RequestStick:
					fmt.Printf("stick requested from %d\n", port)
					<-chopstick
					fmt.Printf("Giving stick on %d\n", port)
					resp := MarshallInts([]int{Ack})
					capture.WriteTo(conn.WriteTo, resp, addr)
				case ExcuseMe:
					if !Excused {
						fmt.Printf("%d has been excused from the table\n", port)
					}
					Excused = true
				}
			}(req)
		}
	}()
	fmt.Printf("server launched")
	return &Philosopher{port, neighbourPort, chopstick, neighbour}
}

//Read incomming udp messages and return the command code and sender address
func getRequest(conn net.PacketConn) (int, net.Addr) {
	var buf [BUFF_SIZE]byte
	_, addr, err := capture.ReadFrom(conn.ReadFrom, buf[0:])
	if err != nil {
		panic(err)
	}
	uArgs := UnmarshallInts(buf)
	return uArgs[0], addr
}

//Transition and print state, then sleep for a random amount of time
func (phil *Philosopher) think() {
	ThinkingState()
	fmt.Printf("%d is thinking.\n", phil.id)
	time.Sleep(time.Duration(rand.Int63n(SLEEP_MAX)))
}

//Eat and then wait for a random amount of time
func (phil *Philosopher) eat() {
	EatingState()
	fmt.Printf("%d is eating.\n", phil.id)
	time.Sleep(time.Duration(rand.Int63n(SLEEP_MAX)))
}

//Attemp to gain a chopstic from a neighbouring philosopher
func (phil *Philosopher) getChopsticks() {
	fmt.Printf("request chopstick %d -> %d\n", phil.id, phil.neighbourId)
	timeout := make(chan bool, 1)
	neighbourChopstick := make(chan bool, 1)
	go func() { time.Sleep(time.Duration(SLEEP_MAX)); timeout <- true }()
	<-phil.chopstick
	LeftChopstickState()
	//timeout function
	fmt.Printf("%v got his chopstick.\n", phil.id)

	//request chopstick function
	go func() {
		//Send Request to Neighbour
		var buf [BUFF_SIZE]byte
		req := MarshallInts([]int{RequestStick})
		conn := phil.neighbour
		capture.Write(conn.Write, req)

		//Read response from Neighbour
		//_, err := phil.neighbour.Read(buf[0:]) TODO cant auto inst
		_, err := capture.Read(conn.Read, buf[0:])
		if err != nil {
			//Connection most likely timed out, chopstick unatainable
			fmt.Printf(err.Error())
			return
		}
		args := UnmarshallInts(buf)
		resp := args[0]
		if resp == Ack {
			fmt.Printf("Received chopstick %d <- %d\n", phil.id, phil.neighbourId)
			neighbourChopstick <- true
			RightChopstickState()
		}
	}()
	select {
	case <-neighbourChopstick:
		fmt.Printf("%v got %v's chopstick.\n", phil.id, phil.neighbourId)
		fmt.Printf("%v has two chopsticks.\n", phil.id)
		return
	case <-timeout:
		fmt.Printf("Timed out on %d\n", phil.id)
		phil.chopstick <- true
		phil.think()
		phil.getChopsticks()
	}
}

func (phil *Philosopher) returnChopsticks() {
	phil.chopstick <- true
	req := MarshallInts([]int{ReleaseStick})
	fmt.Printf("Returning stick %d -> %d\n", phil.id, phil.neighbourId)
	conn := phil.neighbour
	capture.Write(conn.Write, req)
	ThinkingState()
}

func (phil *Philosopher) dine() {
	phil.think()
	phil.getChopsticks()
	phil.eat()
	phil.returnChopsticks()
}

//ask to be excused untill someone says you can
func (phil *Philosopher) leaveTable() {
	for true {
		req := MarshallInts([]int{ExcuseMe})
		conn := phil.neighbour
		capture.Write(conn.Write, req)
		if Excused == true {
			break
		}
	}
}

//main should take as an argument the port number of the philosoper
//and that of its neighbour
func main() {
	var (
		myPort, neighbourPort int
	)
	flag.IntVar(&myPort, "mP", 8080, "The port number this host will listen on")
	flag.IntVar(&neighbourPort, "nP", 8081, "The port this host neighbour will listen on")
	flag.Parse()
	philosopher := makePhilosopher(myPort, neighbourPort)
	for i := 0; i < 100; i++ {
		philosopher.dine()
	}
	fmt.Printf("%v is done dining---------------------------------------------\n", philosopher.id)
	philosopher.leaveTable()
	fmt.Printf("%d left the table --------------------------------------------\n", philosopher.id)
}

//Marshalling Functions

func MarshallInts(args []int) []byte {
	var i, j uint
	marshalled := make([]byte, len(args)*SIZEOFINT, len(args)*SIZEOFINT)
	for j = 0; int(j) < len(args); j++ {
		for i = 0; i < SIZEOFINT; i++ {
			marshalled[(j*SIZEOFINT)+i] = byte(args[j] >> ((SIZEOFINT - 1 - i) * 8))
		}
	}
	return marshalled
}

func UnmarshallInts(args [BUFF_SIZE]byte) []int {
	var i, j uint
	unmarshalled := make([]int, len(args)/SIZEOFINT, len(args)/SIZEOFINT)
	for j = 0; int(j) < len(args)/SIZEOFINT; j++ {
		for i = 0; i < SIZEOFINT; i++ {
			unmarshalled[j] += int(args[SIZEOFINT*(j+1)-1-i] << (i * 8))
		}
	}
	return unmarshalled
}

func PrintErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
