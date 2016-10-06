package dinvRT

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"bitbucket.org/bestchai/dinv/logmerger"
	"github.com/arcaneiceman/GoVector/govec"
	"sort"
	"sync"

)

var (
	initialized = false // Boolean used to track the initalization of the logger
	fast        = true
	id          string        // Timestamp for identifiying loggers
	goVecLogger *govec.GoLog  // GoVec logger, used to track vector timestamps
	packageName string        // TODO packageName is not used -- can it be removed?
	Encoder     *json.Encoder // global name value pair point encoder

	useKV      = true
	resetKV    = true                             // determines if the KV is emptied after the values were written to the log
	varStore   map[string]logmerger.NameValuePair // used to store variable name/value pairs between multiple dumps
	varStoreMx *sync.Mutex                        // manages access to varStore map
	kvDumpIds  []string
	genKVID    func([]logmerger.NameValuePair) string
	
	initMutex *sync.Mutex = &sync.Mutex{}
)

//Dump logs the values of variables passed in as a set of varadic
//arguments. did is the dump id, it must be unique to the dump
//statement, and the host. If a dump statement is constructed by hand,
//the reccomended did is a line number + packagename + port number for
//example 49-api.go-8080. names is a list of variable names logged
//along with the values. names must be formatted as variable name
//comma variable name. The comma is used to split the list and must be
//included. Furthermore the number of names must correspond to the
//number of values or the Dump will panic. Example:
//dinvRT.Dump("49-api.go-8080","counter,variable2,buffersize",counter,variable2,buffersize)
func Dump(did, names string, values ...interface{}) {
	initDinv("")
	nameList := strings.Split(names, ",")
	if len(nameList) != len(values) {
		panic(fmt.Errorf("%s: dump at [%s] has unequal argument lengths", GetId()))
	}
	pairs := make([]logmerger.NameValuePair, len(values))
	p := 0
	for i := 0; i < len(values); i++ {
		if values[i] != nil {
			pairs[p] = newPair(nameList[i], values[i])
			p++
		}
	}
	logPairList(pairs, did)
}

//Track shares the signature and conventions of Dump, but logs to a
//key value store as opposed to directly to disk. The key value store
//is written to disk whenever a host increments is vector time (either
//sending or receiving a message). The purpose of this functionality
//is to create a 1 to 1 correspondiance between logged variables and
//vector timestamps. Track statements are conceptually different than
//dump statements in that they log the values of variables which may
//be out of scope or non existant at the time their values are written
//to disk. Track is intended to capture the summary of a hosts state
//during its transition of vector time.
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
			kvDumpIds = append(kvDumpIds, did) // collect the id of the dump statement being tracked
		}
	}
}

//newPair creates a name value pair out of an arbetrary variable and
//its corresponding name.
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
	if err := Encoder.Encode(point); err != nil {
		fmt.Printf("%s: dinvRT/api.go: Error encoding point: %s", GetId(), err.Error())
		return
	}
}


//variableNamesID merges the names of each variable present in the kv
//store into a unique id based on their name. This merging strategy
//ignores the control flow of a program which collected the variable
//values, it concentrates on the collection of variables insted
func variableNamesID(pairs []logmerger.NameValuePair) string {
	names := make(sort.StringSlice, len(pairs))
	for i := range pairs {
		names[i] = pairs[i].VarName
	}
	names.Sort()
	return logmerger.Hash(concatStrings(names))

}

//smearedDumpID merges the names of the dump statements which where
//reached during the current vector time. The name of the id is each
//of the dumpID's appended together. Duplicates are removed from the
//dumpID's
func smearedDumpID(pairs []logmerger.NameValuePair) string {
	names := make(sort.StringSlice, len(kvDumpIds))
	for i := range kvDumpIds {
		names[i] = kvDumpIds[i]
	}
	//sort the names of the dumps
	names.Sort()
	//remove any duplicate names
	noDups := make(map[string]string, 0)
	for i := range names {
		noDups[names[i]] = names[i]
	}
	uniqueNames := make([]string, 0)
	for i := range noDups {
		uniqueNames = append(uniqueNames, noDups[i])
	}
	kvDumpIds = uniqueNames

	return logmerger.Hash(concatStrings(names))
}

func concatStrings(a []string) string {
	var id string
	for i := range a {
		id += a[i] + "_"
	}
	return id
}

// called from (un)pack functions, so before every network request
// if kv is enabled, all entries in varStore will be logged and the map will be emptied, if resetKV == true
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
	sort.Sort(ByName(pairs))
	kvid := genKVID(pairs)
	logPairList(pairs, kvid)

	if resetKV {
		varStore = make(map[string]logmerger.NameValuePair)
		kvDumpIds = make([]string, 0)
	}
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
	initMutex.Lock()
	defer initMutex.Unlock()
	if !initialized {
		if hostName != "" {
			id = hostName
		} else if os.Getenv("DINV_HOSTNAME") != "" {
			id = os.Getenv("DINV_HOSTNAME")
		} else {
			id = fmt.Sprintf("%d", time.Now().Nanosecond())
		} else {
			id = hostName
		}
		goVecLogger = govec.Initialize(id, id+".log")

		encodedLogname := fmt.Sprintf("%sEncoded.txt", id)

		logFile, err := os.Create(encodedLogname)
		if err != nil {
			panic(fmt.Errorf("%s:dinvRT/api.go: Error creating log file '%s': %s", id, encodedLogname, err.Error()))
		}
		Encoder = json.NewEncoder(logFile)
	}
	if useKV && varStore == nil {
		varStore = make(map[string]logmerger.NameValuePair)
		varStoreMx = &sync.Mutex{}
		//genKVID = variableNamesID
		genKVID = smearedDumpID
		kvDumpIds = make([]string, 0)
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

type ByName []logmerger.NameValuePair

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].VarName < a[j].VarName }
