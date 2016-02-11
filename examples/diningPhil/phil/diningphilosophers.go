package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"bitbucket.org/bestchai/dinv/instrumenter"

	"bitbucket.org/bestchai/dinv/instrumenter/inject"
	"github.com/arcaneiceman/GoVector/govec"
)

const (
	Ack          = 0xFF
	RequestStick = 1
	ReleaseStick = 2
	ExcuseMe     = 3
	SIZEOFINT    = 4
	n            = 50
)

var (
	Eating         bool
	Thinking       bool
	LeftChopstick  bool
	RightChopstick bool
	Excused        bool
	Logger         *govec.GoLog
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
					conn.WriteTo(instrumenter.Pack(resp), addr)
				case ExcuseMe:
					if !Excused {
						fmt.Printf("%d has been excused from the table\n", port)
					}
					Excused = true
				}

				inject.InstrumenterInit("main")
				main_diningphilosophers_113_____vars := []interface{}{n, Thinking, Logger, Ack, RequestStick, ReleaseStick, ExcuseMe, SIZEOFINT, Eating, RightChopstick, LeftChopstick, Excused, req, addr, conn, err, neighbour, errDial, connected, chopstick}
				main_diningphilosophers_113_____varname := []string{"n", "Thinking", "Logger", "Ack", "RequestStick", "ReleaseStick", "ExcuseMe", "SIZEOFINT", "Eating", "RightChopstick", "LeftChopstick", "Excused", "req", "addr", "conn", "err", "neighbour", "errDial", "connected", "chopstick"}
				pmain_diningphilosophers_113____ := inject.CreatePoint(main_diningphilosophers_113_____vars, main_diningphilosophers_113_____varname, "main_diningphilosophers_113____", instrumenter.GetLogger(), instrumenter.GetId())
				inject.Encoder.Encode(pmain_diningphilosophers_113____)

			}(req)
		}
	}()
	fmt.Printf("server launched")
	return &Philosopher{port, neighbourPort, chopstick, neighbour}
}

func getRequest(conn net.PacketConn) (int, net.Addr) {
	var buf [1024]byte
	_, addr, err := conn.ReadFrom(buf[0:])
	if err != nil {
		panic(err)
	}
	args := instrumenter.Unpack(buf[0:]).([]byte)
	//fmt.Printf("Received Request\n")
	uArgs := UnmarshallInts(args)
	return uArgs[0], addr
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

	inject.InstrumenterInit("main")
	main_diningphilosophers_143_____vars := []interface{}{LeftChopstick, Excused, Logger, n, Thinking, ReleaseStick, ExcuseMe, SIZEOFINT, Ack, RequestStick, Eating, RightChopstick}
	main_diningphilosophers_143_____varname := []string{"LeftChopstick", "Excused", "Logger", "n", "Thinking", "ReleaseStick", "ExcuseMe", "SIZEOFINT", "Ack", "RequestStick", "Eating", "RightChopstick"}
	pmain_diningphilosophers_143____ := inject.CreatePoint(main_diningphilosophers_143_____vars, main_diningphilosophers_143_____varname, "main_diningphilosophers_143____", instrumenter.GetLogger(), instrumenter.GetId())
	inject.Encoder.Encode(pmain_diningphilosophers_143____)

}

func (phil *Philosopher) getChopsticks() {
	fmt.Printf("request chopstick %d -> %d\n", phil.id, phil.neighbourId)
	timeout := make(chan bool, 1)
	neighbourChopstick := make(chan bool, 1)
	go func() { time.Sleep(time.Duration(1e9)); timeout <- true }()
	<-phil.chopstick
	LeftChopstickState()
	//timeout function
	fmt.Printf("%v got his chopstick.\n", phil.id)

	//request chopstick function
	go func() {
		//Send Request to Neighbour
		var buf [1024]byte
		req := MarshallInts([]int{RequestStick})
		phil.neighbour.Write(instrumenter.Pack(req))

		//Read response from Neighbour
		_, err := phil.neighbour.Read(buf[0:])
		if err != nil {
			//Connection most likely timed out, chopstick unatainable
			fmt.Printf(err.Error())
			return
		}
		uArgs := instrumenter.Unpack(buf[0:]).([]byte)
		args := UnmarshallInts(uArgs)
		resp := args[0]
		if resp == Ack {
			fmt.Printf("Received chopstick %d <- %d\n", phil.id, phil.neighbourId)
			neighbourChopstick <- true
			RightChopstickState()
		}

		inject.InstrumenterInit("main")
		main_diningphilosophers_178_____vars := []interface{}{LeftChopstick, Excused, n, Thinking, Logger, Ack, RequestStick, ReleaseStick, ExcuseMe, SIZEOFINT, Eating, RightChopstick, buf, req, err, uArgs, args, resp, timeout, neighbourChopstick}
		main_diningphilosophers_178_____varname := []string{"LeftChopstick", "Excused", "n", "Thinking", "Logger", "Ack", "RequestStick", "ReleaseStick", "ExcuseMe", "SIZEOFINT", "Eating", "RightChopstick", "buf", "req", "err", "uArgs", "args", "resp", "timeout", "neighbourChopstick"}
		pmain_diningphilosophers_178____ := inject.CreatePoint(main_diningphilosophers_178_____vars, main_diningphilosophers_178_____varname, "main_diningphilosophers_178____", instrumenter.GetLogger(), instrumenter.GetId())
		inject.Encoder.Encode(pmain_diningphilosophers_178____)

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
	phil.neighbour.Write(instrumenter.Pack(req))
	ThinkingState()

	inject.InstrumenterInit("main")
	main_diningphilosophers_199_____vars := []interface{}{Eating, RightChopstick, LeftChopstick, Excused, n, Thinking, Logger, SIZEOFINT, Ack, RequestStick, ReleaseStick, ExcuseMe, req}
	main_diningphilosophers_199_____varname := []string{"Eating", "RightChopstick", "LeftChopstick", "Excused", "n", "Thinking", "Logger", "SIZEOFINT", "Ack", "RequestStick", "ReleaseStick", "ExcuseMe", "req"}
	pmain_diningphilosophers_199____ := inject.CreatePoint(main_diningphilosophers_199_____vars, main_diningphilosophers_199_____varname, "main_diningphilosophers_199____", instrumenter.GetLogger(), instrumenter.GetId())
	inject.Encoder.Encode(pmain_diningphilosophers_199____)

}

func (phil *Philosopher) dine() {
	phil.think()
	phil.getChopsticks()
	phil.eat()
	phil.returnChopsticks()

	inject.InstrumenterInit("main")
	main_diningphilosophers_207_____vars := []interface{}{RequestStick, ReleaseStick, ExcuseMe, SIZEOFINT, Ack, RightChopstick, Eating, Excused, LeftChopstick, Thinking, Logger, n}
	main_diningphilosophers_207_____varname := []string{"RequestStick", "ReleaseStick", "ExcuseMe", "SIZEOFINT", "Ack", "RightChopstick", "Eating", "Excused", "LeftChopstick", "Thinking", "Logger", "n"}
	pmain_diningphilosophers_207____ := inject.CreatePoint(main_diningphilosophers_207_____vars, main_diningphilosophers_207_____varname, "main_diningphilosophers_207____", instrumenter.GetLogger(), instrumenter.GetId())
	inject.Encoder.Encode(pmain_diningphilosophers_207____)

}

//ask to be excused untill someone says you can
func (phil *Philosopher) leaveTable() {
	for true {
		req := MarshallInts([]int{ExcuseMe})
		phil.neighbour.Write(instrumenter.Pack(req))
		if Excused == true {
			break
		}
	}

	inject.InstrumenterInit("main")
	main_diningphilosophers_219_____vars := []interface{}{LeftChopstick, Excused, n, Thinking, Logger, SIZEOFINT, Ack, RequestStick, ReleaseStick, ExcuseMe, Eating, RightChopstick}
	main_diningphilosophers_219_____varname := []string{"LeftChopstick", "Excused", "n", "Thinking", "Logger", "SIZEOFINT", "Ack", "RequestStick", "ReleaseStick", "ExcuseMe", "Eating", "RightChopstick"}
	pmain_diningphilosophers_219____ := inject.CreatePoint(main_diningphilosophers_219_____vars, main_diningphilosophers_219_____varname, "main_diningphilosophers_219____", instrumenter.GetLogger(), instrumenter.GetId())
	inject.Encoder.Encode(pmain_diningphilosophers_219____)

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
