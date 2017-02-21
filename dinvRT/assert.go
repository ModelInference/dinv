package dinvRT

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"time"
)

// ============================================= CONST =============================================
// Message types
type messageType int

const (
	RTT_REQUEST    messageType = iota
	RTT_RETURN                 = iota
	ASSERT_REQUEST             = iota
	ASSERT_RETURN              = iota
	TIME_REQUEST               = iota
	TIME_RETURN                = iota
	SYNC_REQUEST               = iota
	ASSERT_FAILED              = iota
)

func msgTypeToString(msgT messageType) string {
	switch msgT {
	case RTT_REQUEST:
		return "RTT_REQUEST"
	case RTT_RETURN:
		return "RTT_RETURN"
	case ASSERT_REQUEST:
		return "ASSERT_REQUEST"
	case ASSERT_RETURN:
		return "ASSERT_RETURN"
	case TIME_REQUEST:
		return "TIME_REQUEST"
	case TIME_RETURN:
		return "TIME_RETURN"
	case SYNC_REQUEST:
		return "SYNC_REQUEST"
	case ASSERT_FAILED:
		return "ASSERT_FAILED"
	}
	return "UNDEFINED"
}

// ============================================ STRUCTS ============================================

type assertionFunction func(map[string]map[string]interface{}) bool
type processFunction func(interface{}) interface{}
type nameToValueMap map[string]interface{}

type _message struct {
	MessageType    messageType
	RequestingNode string
	RoundNumber    int
	MessageTime    time.Time
	Result         interface{}
}

// =======================================  GLOBAL VARIABLES =======================================

var address string
var peers []string
var listener *net.UDPConn
var assertableDictionary map[string]interface{}
var assertableFunctions map[string]func(interface{}) interface{}
var roundToResponseMap map[int]*map[string]map[string]interface{}
var roundTripTimeMap map[string]time.Duration
var timingFunction func() time.Time
var rttFunction func(string) time.Duration
var debug = false
var timeOffset = 0 * time.Second

// =======================================  HELPER VARIABLES =======================================

var roundNumber = 0
var roundNumberRTT = 0
var roundNumberTime = 0
var roundTripTime map[string]time.Time
var syncClientTime map[string]time.Time
var syncLocalTime map[string]time.Time

// ========================================  HELPER METHODS ========================================

func checkResult(err error, caller string) {
	if err != nil {
		fmt.Printf("ERROR at %s: %s\n", caller, err)
		os.Exit(-1)
	}
}

func getValue(pointer interface{}) (value reflect.Value) {
	defer func() {
		if recover() != nil {
			fmt.Printf("Zero Value in Assert")
			value = reflect.ValueOf(pointer)
		}
	}()

	value = reflect.ValueOf(pointer).Elem()
	return
}

//Turn an array of integers into a string
func B2S(bs []uint8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

// ===================================== COMMUNICATION METHODS =====================================

func broadcastMessage(payload _message, logMessage string) {
	for _, v := range peers {
		// fmt.Println("Sending to", v)
		go sendToAddr(payload, v, logMessage)
	}
}

func sendToAddr(payload _message, addr string, logMessage string) {
	address, err := net.ResolveUDPAddr("udp", addr)
	checkResult(err, "sendToAddr")

	if debug {
		fmt.Println(logMessage)
		fmt.Printf("Attempting to send [MessageType: %d] to %s\n",
			payload.MessageType, address)
	}
	buf := PackM(payload, logMessage)
	listener.WriteToUDP(buf, address)
}

func receiveConnections() chan _message {
	msg := make(chan _message)

	buf := make([]byte, 8192)

	go func() {
		for {
			n, addr, err := listener.ReadFromUDP(buf[0:])
			var incomingMessage _message
			UnpackM(buf[0:n], &incomingMessage, "Received Message From Node")
			logMessage := fmt.Sprintf("Received message [MessageType: %d] from [%s] of size %d",
				msgTypeToString(incomingMessage.MessageType),
				addr,
				n)
			Local(logMessage)
			if err != nil {
				fmt.Println("READ ERROR: ", err)
				break
			}
			if debug {
				fmt.Printf("Received message [MessageType: %d] from [%s]\n",
					incomingMessage.MessageType,
					addr)
			}
			msg <- incomingMessage
		}
	}()
	return msg
}

func handleAssert(msg _message) {
	msg.MessageType = ASSERT_RETURN
	respondTo := msg.RequestingNode
	msg.RequestingNode = address
	requestedValues := msg.Result.([]interface{})
	valMap := make(map[string]interface{})
	time.Sleep(msg.MessageTime.Sub(getTime()))
	for _, val := range requestedValues {
		intArr := val.([]uint8)
		v := B2S(intArr)
		f, ok := assertableFunctions[v]
		localVal := assertableDictionary[v]
		if ok && f != nil {
			localVal = f(localVal)
		}
		valMap[v] = getValue(localVal)
	}
	msg.Result = valMap
	// fmt.Println(reflect.TypeOf(msg.Result))
	// fmt.Println("handleAssert: Sending to", respondTo)
	sendToAddr(msg, respondTo, "Assert Response")
}

func processData(message_chan chan _message) {
	go func() {
		for {
			message := <-message_chan
			msg_type := message.MessageType
			respondTo := message.RequestingNode

			// Switch on the message type byte, each case should do it's own parsing with the buffer
			switch msg_type {
			case RTT_REQUEST:
				message.MessageType = RTT_RETURN
				message.RequestingNode = address
				// fmt.Println("RTT_REQUEST: Sending to", respondTo)
				sendToAddr(message, respondTo, "Round Trip Response")
				break
			case RTT_RETURN:
				if roundNumberRTT == message.RoundNumber {
					roundTripTimeMap[message.RequestingNode] = getTime().Sub(roundTripTime[message.RequestingNode])
				}
				break
			case ASSERT_REQUEST:
				go handleAssert(message)
				break
			case ASSERT_RETURN:
				val, ok := roundToResponseMap[message.RoundNumber]
				if ok {
					roundMap := *val
					returnedValues := message.Result.(map[interface{}]interface{})
					//fmt.Println(returnedValues)
					returnedValuesCopy := make(map[string]interface{})
					for k, v := range returnedValues {
						returnedValuesCopy[k.(string)] = v
					}
					roundMap[message.RequestingNode] = returnedValuesCopy
				}
				break
			case TIME_REQUEST:
				if roundNumberTime <= message.RoundNumber {
					roundNumberTime = message.RoundNumber
					message.MessageType = TIME_RETURN
					message.RequestingNode = address
					message.MessageTime = getTime()
					// fmt.Println("TIME_REQUEST: Sending to", respondTo)
					sendToAddr(message, respondTo, "Round Trip Response")
				}
				break
			case TIME_RETURN:
				if roundNumberTime == message.RoundNumber {
					syncClientTime[message.RequestingNode] = roundTripTime[message.RequestingNode]
					syncLocalTime[message.RequestingNode] = time.Time{}.Add(syncLocalTime[message.RequestingNode].Add(getTime().Sub(time.Time{})).Sub(time.Time{}) / 2)
				}
				break
			case SYNC_REQUEST:
				if roundNumberTime <= message.RoundNumber {
					roundNumberTime = message.RoundNumber
					result, err := message.Result.(int64)
					if err {
						// fmt.Printf("Result: %v, parsed: %d, error: %v\n", message.Result, result, err)
					} else {
						timeOffset = timeOffset + time.Duration(result)
					}
					// fmt.Printf("Time is: %v, time offset: %v\n", getTime(), timeOffset)
				}
				break
			case ASSERT_FAILED:
				// fmt.Println("ASSERTION FAILED")
				Local("Received ASSERTION FAILED")
				// time.Sleep(time.Second)
				os.Exit(-1)
			default:
				fmt.Printf("Error: unknown message type received [%d]\n", msg_type)
			}
		}
	}()
}

// ===================================== RTT METHODS =====================================

func getRTT(addr string) {
	RTTmessage := _message{MessageType: RTT_REQUEST, RequestingNode: address, RoundNumber: roundNumberRTT}
	roundTripTime[addr] = getTime()
	// fmt.Println("getRTT: Sending to", addr)
	sendToAddr(RTTmessage, addr, "Round Trip Request")
}

func handleRTT() {
	go func() {
		for {
			// TODO: This can probably be configurable to not flood the network
			time.Sleep(5 * time.Second)
			roundNumberRTT++
			for _, v := range peers {
				getRTT(v)
			}
		}
	}()
}

func GetAssertDelay() time.Duration {
	duration := 0 * time.Second
	for _, v := range roundTripTimeMap {
		if v > duration {
			duration = v
		}
	}
	message := fmt.Sprintf("RTT: %v", 50*time.Millisecond)
	Local(message)
	return 10 * time.Millisecond //+ duration
}

// ===================================== TIMING METHODS =====================================

func getTime() time.Time {
	return time.Now().Add(timeOffset)
}

func syncTime(addr string) {
	RTTmessage := _message{MessageType: TIME_REQUEST, RequestingNode: address, RoundNumber: roundNumberTime}
	syncLocalTime[addr] = getTime()
	// fmt.Println("syncTime: Sending to", addr)
	sendToAddr(RTTmessage, addr, "Get Time Request")
}

func sendDiffTime(addr string) {
	RTTmessage := _message{MessageType: SYNC_REQUEST, RequestingNode: address, RoundNumber: roundNumberTime, Result: syncClientTime[addr].Sub(syncLocalTime[addr])}
	// fmt.Println("sendDiffTime: Sending to", addr)
	sendToAddr(RTTmessage, addr, "Sync Time Request")
}

func handleTimeSync() {
	go func() {
		for {
			time.Sleep(4 * time.Second)
			for _, v := range peers {
				delete(syncClientTime, v)
				syncTime(v)
			}
			time.Sleep(2 * time.Second)
			roundNumberTime++
			for k, _ := range syncClientTime {
				sendDiffTime(k)
			}
		}
	}()
}

// =======================================  PUBLIC METHODS =======================================
//Listening address
//option 1) addr is the format :portnumber
//option 2) environment variable DINV_ASSERT_LISTEN is the format
//ip:port
//Peers
//option 1) assertPeers is a string array of the format :ports
//option 2) enviornment variable DINV_ASSERT_PEERS is the form
//ip:port,ip:port,...,ip:port which may or may not include the
//peer itself
func InitDistributedAssert(addr string, assertPeers []string, processName string) {
	//initalize dinvRT if it is not allready running
	initDinv("")
	//initalize asserting address
	if addr != "" {
		address = addr
	} else if os.Getenv("DINV_ASSERT_LISTEN") != "" {
		address = os.Getenv("DINV_ASSERT_LISTEN")
	} else {
		goVecLogger.LogLocalEvent("Cannot Initalize Assert Address Dying!")
		os.Exit(1)
	}
	//initalize assert peers
	if assertPeers != nil {
		peers = assertPeers
	} else if os.Getenv("DINV_ASSERT_PEERS") != "" {
		peers = strings.Split(os.Getenv("DINV_ASSERT_PEERS"), ",")
	} else {
		goVecLogger.LogLocalEvent("Cannot parse assert peers dying!!")
		os.Exit(1)
	}
	//remove self from peer list
	tmpNeighbours := make([]string, 0)
	for _, n := range peers {
		if n != address {
			tmpNeighbours = append(tmpNeighbours, n)
		}
	}
	peers = tmpNeighbours

	listen_address, err := net.ResolveUDPAddr("udp4", address)
	// fmt.Println("Listening on address: ", address)
	listener, err = net.ListenUDP("udp4", listen_address)
	if listener == nil {
		fmt.Println("Error could not listen on ", address)
		fmt.Println("Error: ", err)
		os.Exit(-1)
	}

	assertableDictionary = make(map[string]interface{})
	assertableFunctions = make(map[string]func(interface{}) interface{})

	message := receiveConnections()

	if debug {
		fmt.Println("Calling process data")
	}

	processData(message)

	syncClientTime = make(map[string]time.Time)
	syncLocalTime = make(map[string]time.Time)
	roundTripTime = make(map[string]time.Time)
	roundTripTimeMap = make(map[string]time.Duration)
	roundToResponseMap = make(map[int]*map[string]map[string]interface{})

	lowest := true
	for _, v := range peers {
		roundTripTimeMap[v] = time.Second
		if v < addr {
			lowest = false
		}
	}

	if lowest {
		handleTimeSync()
	}
	handleRTT()
}

func AddAssertable(name string, pointer interface{}, f processFunction) {
	if reflect.TypeOf(pointer).Kind() != reflect.Ptr {
		fmt.Printf("Error: Tried adding %s as variable, did not pass pointer!\n", name)
		os.Exit(-1)
	}
	assertableDictionary[name] = pointer
	assertableFunctions[name] = f
	// fmt.Printf("%s %s: %v\n", address, name, getValue(pointer))
}

func Assert(outerFunc func(map[string]map[string]interface{}) bool, requestedValues map[string][]string) {
	f := assertionFunction(outerFunc)
	localRoundNumber := roundNumber
	roundNumber++

	maxRTT := GetAssertDelay()
	responseMap := make(map[string]map[string]interface{})
	roundToResponseMap[localRoundNumber] = &responseMap

	assertTime := getTime()
	assertTime = assertTime.Add(maxRTT)
	for k, v := range requestedValues {
		AssertRequestMessage := _message{MessageType: ASSERT_REQUEST, RequestingNode: address, RoundNumber: localRoundNumber, MessageTime: assertTime, Result: v}
		// fmt.Println("AssertRequest: Sending to", k)
		go sendToAddr(AssertRequestMessage, k, "Requesting Assertion")
	}

	time.Sleep(2 * maxRTT)
	delete(roundToResponseMap, localRoundNumber)

	if !f(responseMap) {
		// fmt.Println("ASSERTION FAILED: ", responseMap)
		message := fmt.Sprintf("ASSERTION FAILED: %#+v", responseMap)
		Local(message)
		for k, _ := range requestedValues {
			AssertFailedMessage := _message{MessageType: ASSERT_FAILED, RequestingNode: address, RoundNumber: localRoundNumber, MessageTime: assertTime}
			// fmt.Println("Attempting to send fail message")
			// fmt.Println("AssertFailed: Sending to", k)
			sendToAddr(AssertFailedMessage, k, "Assertion Failed")
		}
		time.Sleep(maxRTT)
		os.Exit(-1)
	} else {
		message := fmt.Sprintf("ASSERTION PASSED: %#+v", responseMap)
		Local(message)
	}
}

func GetPeers() []string {
	return peers
}
