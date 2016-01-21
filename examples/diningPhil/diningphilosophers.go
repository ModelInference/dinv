package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

const (
	Ack          = 0xFF
	RequestStick = 1
	ReleaseStick = 2
	SIZEOFINT    = 4
	n            = 50
)

var (
	Eating         bool
	Thinking       bool
	LeftChopstick  bool
	RightChopstick bool
)

func EatingState() {
	Eating = true
	Thinking = false
	LeftChopstick = true
	RightChopstick = true
}

func ThinkingState() {
	Eating = false
	Thinking = true
	LeftChopstick = false
	RightChopstick = false
}

func LeftChopstickState() {
	Eating = false
	Thinking = true
	LeftChopstick = true
}

func RightChopstickState() {
	Eating = false
	Thinking = true
	RightChopstick = true
}

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
	fmt.Printf("listening on %d\n", port)
	var neighbour *net.UDPConn
	var errDial error
	//setup connection to linn server
	connected := false
	for !connected {
		neighbourAddr, errR := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", neighbourPort))
		PrintErr(errR)
		//TODO sending port set to +1000 to restrict command line
		//arguments, this should be handled better
		listenAddr, errL := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", port+1000))
		PrintErr(errL)
		neighbour, errDial = net.DialUDP("udp", listenAddr, neighbourAddr)
		PrintErr(errDial)
		connected = (errR == nil) && (errL == nil) && (errDial == nil)
	}

	chopstick := make(chan bool, 1)
	chopstick <- true
	fmt.Printf("launching chopstick server\n")
	go func() {
		defer fmt.Printf("Chopstick #%d\n is down", port) //attempt to show when the chopsticks are no longer available
		for true {
			var buf [1024]byte
			_, addr, err := conn.ReadFrom(buf[0:])
			if err != nil {
				panic(err)
			}
			fmt.Printf("Received Request\n")
			uArgs := UnmarshallInts(buf[0:])
			req := uArgs[0]
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
					conn.WriteTo(resp, addr)
				}
			}(req)
		}
	}()
	fmt.Printf("server launched")
	return &Philosopher{port, neighbourPort, chopstick, neighbour}
}

func (phil *Philosopher) think() {
	ThinkingState()
	fmt.Printf("%d is thinking.\n", phil.id)
	time.Sleep(time.Duration(rand.Int63n(1e9)))
}

func (phil *Philosopher) eat() {
	EatingState()
	fmt.Printf("%d is eating.\n", phil.id)
	time.Sleep(time.Duration(rand.Int63n(1e9)))
}

func (phil *Philosopher) getChopsticks() {
	fmt.Printf("request chopstick %d -> %d\n", phil.id, phil.neighbourId)
	timeout := make(chan bool, 1)
	neighbourChopstick := make(chan bool, 1)
	//request chopstick function
	go func() { time.Sleep(time.Duration(1e9)); timeout <- true }()
	<-phil.chopstick
	LeftChopstickState()
	//timeout function
	fmt.Printf("%v got his chopstick.\n", phil.id)

	go func() {
		var buf [1024]byte
		req := MarshallInts([]int{RequestStick})
		phil.neighbour.Write(req)
		phil.neighbour.Read(buf[0:])
		args := UnmarshallInts(buf[0:])
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
	phil.neighbour.Write(req)
	ThinkingState()
}

func (phil *Philosopher) dine() {
	phil.think()
	phil.getChopsticks()
	phil.eat()
	phil.returnChopsticks()
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
	philosopher.dine()
	fmt.Printf("%v is done dining\n", philosopher.id)
	time.Sleep(time.Duration(1000))
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

func UnmarshallInts(args []byte) []int {
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

/*
old main function used when all the hosts were mannaged by a single go program
func main() {
	philosophers := make([]*Philosopher, n)
	for i := 0; i < n; i++ {
		philosophers[i] = makePhilosopher(i, (i+1)%n)
	}
	setup := false
	fmt.Printf("settingUp\n")
	for !setup {
		setup = true
		for i := 0; i < n; i++ {
			if philosophers[i] == nil {
				//fmt.Printf("%d", i)
				setup = false
			}
		}
	}
	fmt.Printf("There are %v philosophers sitting at a table.\n", len(philosophers))
	fmt.Println("They each have one chopstick, and must borrow from their neighbor to eat.")
	announce := make(chan *Philosopher)
	for _, phil := range philosophers {
		go phil.dine(announce)
	}
	for i := 0; i < len(philosophers); i++ {
		phil := <-announce
		fmt.Printf("%v is done dining %d/%d.\n", phil.id, i, n-1)
	}
}
*/
