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
	initialized = false
	stamp       int
	goVecLogger *govec.GoLog
)

func Pack(msg []byte) []byte {
	initDinv()
	id := getCallingFunctionID()
	return goVecLogger.PrepareSend("Sending from "+id+" "+fmt.Sprintf("%d", stamp), msg)

}

func Unpack(msg []byte) []byte {
	initDinv()
	id := getCallingFunctionID()
	return goVecLogger.UnpackReceive("Received on "+id+" "+fmt.Sprintf("%d", stamp), msg)
}

func GetLogger() *govec.GoLog {
	initDinv()
	return goVecLogger
}

func GetStamp() int {
	initDinv()
	return stamp
}

func initDinv() {
	if !initialized {
		stamp = time.Now().Nanosecond()
		stampString := fmt.Sprintf("%d", stamp)
		goVecLogger = govec.Initialize(stampString, stampString+".log")
		initialized = true
	}
}

func getCallingFunctionID() string {
	profiles := pprof.Profiles()
	block := profiles[1]
	var buf bytes.Buffer
	block.WriteTo(&buf, 1)
	passedFrontOnStack := false
	re := regexp.MustCompile("([a-zA-Z0-9]+.go:[0-9]+)")
	ownFilename := regexp.MustCompile("user.go") // hardcoded own filename
	matches := re.FindAllString(fmt.Sprintf("%s", buf), -1)
	for _, match := range matches {
		if passedFrontOnStack && !ownFilename.MatchString(match) {
			return match
		} else if ownFilename.MatchString(match) {
			passedFrontOnStack = true
		}
		fmt.Printf("found %s\n", match)
	}
	fmt.Printf("%s\n", buf)
	return ""
}
