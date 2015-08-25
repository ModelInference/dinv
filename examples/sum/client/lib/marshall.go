package client

import (
	"fmt"
	"os"
	"bitbucket.org/bestchai/dinv/instrumenter/inject"
	"bitbucket.org/bestchai/dinv/instrumenter"
)

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func MarshallInts(args []int) []byte {
	var i, j uint
	marshalled := make([]byte, len(args)*SIZEOFINT, len(args)*SIZEOFINT)
	for j = 0; int(j) < len(args); j++ {
		for i = 0; i < SIZEOFINT; i++ {
			marshalled[(j*SIZEOFINT)+i] = byte(args[j] >> ((SIZEOFINT - 1 - i) * 8))
		}
	}
	
inject.InstrumenterInit("client")
client_marshall_23_____vars := []interface{}{i,j,marshalled}
client_marshall_23_____varname := []string{"i","j","marshalled"}
pclient_marshall_23____ := inject.CreatePoint(client_marshall_23_____vars, client_marshall_23_____varname,"client_marshall_23____",instrumenter.GetLogger(),instrumenter.GetId())
inject.Encoder.Encode(pclient_marshall_23____)

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
