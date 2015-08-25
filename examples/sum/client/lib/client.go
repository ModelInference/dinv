package client

import (
	"fmt"
	"math/rand"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/instrumenter"
	"bitbucket.org/bestchai/dinv/instrumenter/inject"
)

const (
	SIZEOFINT	= 4
	ADDITION_ARGS	= 2
	LARGEST_TERM	= 100
	RUNS		= 500
)

var (
	buf				[1024]byte
	term1, term2, sum		int
	thegreatestFloatOfthemAll	float32
)

func Client() {
	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)
	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)
	instrumenter.Initalize("The greatest client of them all")
	//dump

	for t := 0; t < RUNS; t++ {
		term1, term2 = rand.Int()%LARGEST_TERM, rand.Int()%LARGEST_TERM

		msg := MarshallInts([]int{term1, term2})
		// sending UDP packet to specified address and port
		_, errWrite := conn.Write(instrumenter.Pack(msg))

		
inject.InstrumenterInit("client")
client_client_42_____vars := []interface{}{ADDITION_ARGS,RUNS,term2,thegreatestFloatOfthemAll,SIZEOFINT,LARGEST_TERM,buf,term1,sum}
client_client_42_____varname := []string{"ADDITION_ARGS","RUNS","term2","thegreatestFloatOfthemAll","SIZEOFINT","LARGEST_TERM","buf","term1","sum"}
pclient_client_42____ := inject.CreatePoint(client_client_42_____vars, client_client_42_____varname,"client_client_42____",instrumenter.GetLogger(),instrumenter.GetId())
inject.Encoder.Encode(pclient_client_42____)

		printErr(errWrite)
		//adding local events for testing lattice /jan 23 2015
		//		for i := 0; i < 3; i++ {
		//			Logger.LogLocalEvent("Twittle Thumbs")
		//		}
		// Reading the response message

		_, errRead := conn.Read(buf[0:])
		ret := instrumenter.Unpack(buf[0:]).([]byte)
		printErr(errRead)

		uret := UnmarshallInts(ret)
		sum = uret[0]
		thegreatestFloatOfthemAll = float32(sum)
		
inject.InstrumenterInit("client")
client_client_57_____vars := []interface{}{SIZEOFINT,LARGEST_TERM,buf,term1,sum,ADDITION_ARGS,RUNS,term2,thegreatestFloatOfthemAll,rAddr,errR,lAddr,errL,conn,errDial}
client_client_57_____varname := []string{"SIZEOFINT","LARGEST_TERM","buf","term1","sum","ADDITION_ARGS","RUNS","term2","thegreatestFloatOfthemAll","rAddr","errR","lAddr","errL","conn","errDial"}
pclient_client_57____ := inject.CreatePoint(client_client_57_____vars, client_client_57_____varname,"client_client_57____",instrumenter.GetLogger(),instrumenter.GetId())
inject.Encoder.Encode(pclient_client_57____)

		fmt.Printf("\rExecuting %3.0f%%", float32(t)/float32(RUNS)*100)
		sum = 0
	}
	fmt.Println()
	os.Exit(0)
}
