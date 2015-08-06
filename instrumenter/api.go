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

	"github.com/wantonsolutions/GoVector/govec"
)

var (
	initialized = false      //Boolean used to track the initalization of the logger
	stamp       int          //Timestamp for identifiying loggers
	goVecLogger *govec.GoLog //GoVec logger, used to track vector timestamps
)

//Pack takes an an argument a set of bytes msg, and returns that set
//of bytes with all current logging information wrapping the buffer.
//This method is to be used on all data prior to communcition
//PostCondition the byte array contains all current logging
//information
func Pack(msg []byte) []byte {
	initDinv()
	id := getCallingFunctionID()
	return goVecLogger.PrepareSend("Sending from "+id+" "+fmt.Sprintf("%d", stamp), msg)

}

//Unpack removes logging information from an array of bytes. The bytes
//are returned with the logging info removed.
//This method is to be used on all data upon receving it
//Precondition, the array of bytes was packed before sending
func Unpack(msg []byte) []byte {
	initDinv()
	id := getCallingFunctionID()
	return goVecLogger.UnpackReceive("Received on "+id+" "+fmt.Sprintf("%d", stamp), msg)
}

//GetLogger is used to retreive the logger maintaining a vector
//timestamp.
//This function is called by code injected at //@dump
//annotations and is not recomened for general use, but is available
//for debugging.
func GetLogger() *govec.GoLog {
	initDinv()
	return goVecLogger
}

//GetStap returns the ID of the current logger
//TODO this function is an artifact of the timestap identification
//system July-2015 and should be removed when a more robust strategy
//is implemented
func GetStamp() int {
	initDinv()
	return stamp
}

//initDinv instatiates a logger for the running process, and generates
//an id for it. This method is called only once per logger, and
//writes the first log.
func initDinv() {
	if !initialized {
		stamp = time.Now().Nanosecond()
		stampString := fmt.Sprintf("%d", stamp)
		goVecLogger = govec.Initialize(stampString, stampString+".log")
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
