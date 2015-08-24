//api provides a set of functions for analyzing network traffic. The
//Pack() and Unpack() functions are the primary interface for
//tracking communction. They must be used on all transmitted data
//before and after transmission respecfully.

package instrumenter

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime/pprof"
	"time"

	"github.com/arcaneiceman/GoVector/govec"
)

var (
	initialized = false      //Boolean used to track the initalization of the logger
	id          string       //Timestamp for identifiying loggers
	goVecLogger *govec.GoLog //GoVec logger, used to track vector timestamps
)

//Pack takes an an argument a set of bytes msg, and returns that set
//of bytes with all current logging information wrapping the buffer.
//This method is to be used on all data prior to communcition
//PostCondition the byte array contains all current logging
//information
func Pack(msg interface{}) []byte {
	initDinv("")
	callingFunction := getCallingFunctionID()
	return goVecLogger.PrepareSend("Sending from "+callingFunction+" "+id, msg)

}

//Unpack removes logging information from an array of bytes. The bytes
//are returned with the logging info removed.
//This method is to be used on all data upon receving it
//Precondition, the array of bytes was packed before sending
func Unpack(msg []byte) interface{} {
	initDinv("")
	callingFunction := getCallingFunctionID()
	return goVecLogger.UnpackReceive("Received on "+callingFunction+" "+id, msg)
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
func CustomEncoderDecoder(encoder func(interface{}) ([]byte, error), decoder func([]byte) (interface{}, error)) {
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
		initialized = true
	}
}

//getCallingFunctionID returns the file name and line number of the
//program which called api.go. This function is used to generate
//logging statements dynamically.
func getCallingFunctionID() string {
	profiles := pprof.Profiles()
	block := profiles[1]
	var buf bytes.Buffer
	block.WriteTo(&buf, 1)
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
