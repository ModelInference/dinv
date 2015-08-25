package server

import (
	"fmt"
	"net"
	"os"

	"bitbucket.org/bestchai/dinv/instrumenter"
	"bitbucket.org/bestchai/dinv/instrumenter/inject"
)

const SIZEOFINT = 4

var (
	buf			[1024]byte
	term1, term2, sum	int
)

func Server() {
	conn, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	printErr(err)

	//main loop
	for {
		if err != nil {
			printErr(err)
			continue
		}
		handleConn(conn)
		//fmt.Println("some one connected!")
	}
	conn.Close()
}

func handleConn(conn net.PacketConn) {

	_, addr, err := conn.ReadFrom(buf[0:])
	args := instrumenter.Unpack(buf[0:]).([]byte)
	printErr(err)
	
inject.InstrumenterInit("server")
server_server_43_____vars := []interface{}{SIZEOFINT,buf,term1,term2,sum}
server_server_43_____varname := []string{"SIZEOFINT","buf","term1","term2","sum"}
pserver_server_43____ := inject.CreatePoint(server_server_43_____vars, server_server_43_____varname,"server_server_43____",instrumenter.GetLogger(),instrumenter.GetId())
inject.Encoder.Encode(pserver_server_43____)

	//fmt.Printf("recieved: %s of size %d, with args %d", buf, n, args)

	//adding local events for testing lattice /jan 23 2015
	//	for i := 0; i < 3; i++ {
	//		Logger.LogLocalEvent("Twittle Thumbs")
	//	}
	uArgs := UnmarshallInts(args)
	term1, term2 = uArgs[0], uArgs[1]
	sum = term1 + term2
	msg := MarshallInts([]int{sum})
	conn.WriteTo(instrumenter.Pack(msg), addr)
	
inject.InstrumenterInit("server")
server_server_55___sending_to_client___vars := []interface{}{term1,term2,sum,SIZEOFINT,buf}
server_server_55___sending_to_client___varname := []string{"term1","term2","sum","SIZEOFINT","buf"}
pserver_server_55___sending_to_client__ := inject.CreatePoint(server_server_55___sending_to_client___vars, server_server_55___sending_to_client___varname,"server_server_55___sending_to_client__",instrumenter.GetLogger(),instrumenter.GetId())
inject.Encoder.Encode(pserver_server_55___sending_to_client__)

}
