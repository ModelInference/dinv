package dinvRT

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"bitbucket.org/bestchai/dinv/logmerger"
	"github.com/arcaneiceman/GoVector/govec"
	"sync"
)

var (
	initialized = false //Boolean used to track the initalization of the logger
	fast        = true
	id          string                             //Timestamp for identifiying loggers
	goVecLogger *govec.GoLog                       //GoVec logger, used to track vector timestamps
	packageName string                             // TODO packageName is not used -- can it be removed?
	useKV       bool                               // set to true if $USE_KV is set to any value
	Encoder     *gob.Encoder                       // global name value pair point encoder
	varStore    map[string]logmerger.NameValuePair // used to store variable name/value pairs between multiple dumps
	varStoreMx  *sync.Mutex                        // manages access to varStore map
)

func Dump(did, names string, values ...interface{}) {
	initDinv("")
	nameList := strings.Split(names, ",")
	if len(nameList) != len(values) {
		panic(fmt.Errorf("dump at [%s] has unequal argument lengths"))
	}
	pairs := make([]logmerger.NameValuePair, 0, len(values))
	for i := 0; i < len(values); i++ {
		if values[i] != nil {
			pairs = append(pairs, newPair(nameList[i], values[i]))
		}
	}
	logPairList(pairs, did)

}

func Track(did, names string, values ...interface{}) {
	useKV = true
	initDinv("")
	nameList := strings.Split(names, ",")
	if len(nameList) != len(values) {
		panic(fmt.Errorf("track at [%s] has unequal argument lengths"))
	}
	varStoreMx.Lock()
	defer varStoreMx.Unlock()
	for i := 0; i < len(values); i++ {
		if values[i] != nil {
			varStore[nameList[i]] = newPair(nameList[i], values[i])
		}
	}
}

func newPair(name string, value interface{}) (pair logmerger.NameValuePair) {
	pair = logmerger.NameValuePair{VarName: name, Value: value, Type: ""}
	//nasty switch statement for catching most basic go types
	switch reflect.TypeOf(value).Kind() {
	case reflect.Bool:
		pair.Type = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		pair.Type = "int"
	case reflect.Float32, reflect.Float64:
		pair.Type = "float"
	case reflect.String:
		pair.Type = "string"
		//unknown type to daikon don't add the variable
		// TODO: should a variable with unknown type still be included?
	}
	return pair
}

// write array of variables to log
func logPairList(pairs []logmerger.NameValuePair, did string) {
	var dumpID string
	if fast || did != "" {
		dumpID = did
	} else {
		dumpID = getHashedId()
	}

	point := logmerger.Point{
		Dump:               pairs,
		Id:                 dumpID,
		VectorClock:        GetLogger().GetCurrentVC(),
		CommunicationDelta: 0,
	}
	// fmt.Printf("%v", point)
	Encoder.Encode(point)
}

// called from (un)pack functions, so before every network request
// if kv is enabled, all entries in varStore will be logged and the map will be emptied
func logVarStore() {
	if !useKV {
		return
	}
	varStoreMx.Lock()
	defer varStoreMx.Unlock()
	pairs := make([]logmerger.NameValuePair, 0, len(varStore))
	for _, pair := range varStore {
		pairs = append(pairs, pair)
	}
	logPairList(pairs, "kv")
	varStore = make(map[string]logmerger.NameValuePair)
}

//Pack takes an an argument a set of bytes msg, and returns that set
//of bytes with all current logging information wrapping the buffer.
//This method is to be used on all data prior to communcition
//PostCondition the byte array contains all current logging
//information
func Pack(msg interface{}) []byte {
	initDinv("")
	logVarStore()
	if fast {
		return goVecLogger.PrepareSend("Sending from "+id, msg)
	} else {
		return goVecLogger.PrepareSend("Sending from "+getCallingFunctionID()+" "+id, msg) // slow
	}
}

//PackM operates identically to Pack, but allows for custom messages
//to be logged
func PackM(msg interface{}, log string) []byte {
	initDinv("")
	logVarStore()
	return goVecLogger.PrepareSend(log, msg)
}

//Unpack removes logging information from an array of bytes. The bytes
//are returned with the logging info removed.
//This method is to be used on all data upon receving it
//Precondition, the array of bytes was packed before sending
func Unpack(msg []byte, pack interface{}) {
	initDinv("")
	logVarStore()
	if fast {
		goVecLogger.UnpackReceive("Received on "+id, msg, pack)
	} else {
		goVecLogger.UnpackReceive("Received on "+getCallingFunctionID()+" "+id, msg, pack) //Slow
	}
	return
}

func UnpackM(msg []byte, pack interface{}, log string) {
	initDinv("")
	logVarStore()
	goVecLogger.UnpackReceive(log, msg, pack)
	return
}

//Initalize is an optional call for naming hosts uniquely based on a
//user specified string
func Initalize(hostName string) error {
	if initialized == true {
		return fmt.Errorf("Dinv logger has allready been initalized. Initalize must be the first call to dinv's api, including dump statements")
	} else {
		//remove whitespace from host name, makes parsing later easier
		whiteSpace := regexp.MustCompile("\\s")
		hostName = whiteSpace.ReplaceAllString(hostName, "_")
		initDinv(hostName)
	}
	return nil
}

//GetLogger is used to retreive the logger maintaining a vector
//timestamp.
//This function is called by code injected at //@dump
//annotations and is not recomened for general use, but is available
//for debugging.
func GetLogger() *govec.GoLog {
	initDinv("")
	return goVecLogger
}

//GetId returns the ID of the current logger
//TODO this function is an artifact of the timestap identification
//system July-2015 and should be removed when a more robust strategy
//is implemented
func GetId() string {
	initDinv("")
	return id
}

//CustomEncoderDecoder allows users to specify the functions that are
//used to encode or decode their messages. In cases where message
//types are not basic Go types this can be useful
//encoder is a function taking an interface as an argument, and
//returning that interface as an encoded array of bytes, return nil if
//the interface is not encodeable
//decoder is encoders counterpart, taking an encoded array of bytes
//and returning the underlying go object as an interface. The returned
//value must by type cast before it can be used
func CustomEncoderDecoder(encoder func(interface{}) ([]byte, error), decoder func([]byte, interface{}) error) {
	initDinv("")
	gvLogger := GetLogger()
	gvLogger.SetEncoderDecoder(encoder, decoder)
}

//initDinv instatiates a logger for the running process, and generates
//an id for it. This method is called only once per logger, and
//writes the first log.
func initDinv(hostName string) {
	if !initialized {
		if hostName == "" {
			id = fmt.Sprintf("%d", time.Now().Nanosecond())
		} else {
			id = hostName
		}
		goVecLogger = govec.Initialize(id, id+".log")

		encodedLogname := fmt.Sprintf("%sEncoded.txt", id)
		encodedLog, _ := os.Create(encodedLogname)
		Encoder = gob.NewEncoder(encodedLog)

	}
	if useKV && varStore == nil {
		varStore = make(map[string]logmerger.NameValuePair)
		varStoreMx = &sync.Mutex{}
	}

	initialized = true
}

func getHashedId() string {
	return GetId() + "_" + getCallingFunctionID()
}

//getCallingFunctionID returns the file name and line number of the
//program which called api.go. This function is used to generate
//logging statements dynamically.
func getCallingFunctionID() string {
	profiles := pprof.Profiles()
	block := profiles[1]
	var buf bytes.Buffer
	block.WriteTo(&buf, 1)
	//fmt.Printf("%s",buf)
	passedFrontOnStack := false
	re := regexp.MustCompile("([a-zA-Z0-9]+.go:[0-9]+)")
	ownFilename := regexp.MustCompile("api.go") // hardcoded own filename
	matches := re.FindAllString(fmt.Sprintf("%s", buf), -1)
	for _, match := range matches {
		if passedFrontOnStack && !ownFilename.MatchString(match) {
			return match
		} else if ownFilename.MatchString(match) {
			passedFrontOnStack = true
		}
		//fmt.Printf("found %s\n", match)
	}
	fmt.Printf("%s\n", buf)
	return ""
}

/* Injection Code */
//injection code is used to dynamicly write an injection file,
//containing methods called by dump statements

//header_code contains all the needed imports for the injection code,
//and is designed to have the package name written at runtime

//body code contains utility functions called by the code injected at
//dump statements
//TODO add comments to the inject code
//TODO build array of acceptable types for encoding
//TODO make the logger an argument to CreatePoint

func CreatePoint(vars []interface{}, varNames []string, id string, logger *govec.GoLog, hash string) logmerger.Point {
	numVars := len(varNames)
	dumps := make([]logmerger.NameValuePair, 0)
	hashedId := hash + "_" + id
	for i := 0; i < numVars; i++ {
		if vars[i] != nil {
			//nasty switch statement for catching most basic go types
			var dump logmerger.NameValuePair
			dump.VarName = varNames[i]
			dump.Value = vars[i]
			switch reflect.TypeOf(vars[i]).Kind() {
			case reflect.Bool:
				dump.Type = "boolean"
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				dump.Type = "int"
			case reflect.Float32, reflect.Float64:
				dump.Type = "float"
			case reflect.String:
				dump.Type = "string"
			//unknown type to daikon don't add the variable
			default:
				continue
			}
			dumps = append(dumps, dump)
		}
	}
	point := logmerger.Point{dumps, hashedId, logger.GetCurrentVC(), 0}
	return point
}

func Local(logger *govec.GoLog, id string) {
	logger.LogLocalEvent(fmt.Sprintf("Dump @ id %s", id))
}
