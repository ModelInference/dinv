package chainkv

import (
	"fmt"
	"net"
//	"os"
//	"time"

	"github.com/arcaneiceman/GoVector/govec"
)




func Node(id, next, last string) {
	Logger := govec.Initialize("Node-"+id, "Node-"+id)
	myAddr, err := net.ResolveUDPAddr("udp", ":"+id)
	listen, err := net.ListenUDP("udp", myAddr)
	printErr(err)
	nextNode, err := net.ResolveUDPAddr("udp", ":"+next)
	var buf [512]byte
	
	for {
		_, _, err := listen.ReadFrom(buf[0:])
		printErr(err)
		var incommingMessage int
		Logger.UnpackReceive("Received Message From Previous Node", buf[0:], &incommingMessage)

		fmt.Printf("Recieved %d\n", incommingMessage)
		replicate(incommingMessage)
		
		outBuf := Logger.PrepareSend("Sending message to Next Node", incommingMessage)
		_, errWrite := listen.WriteToUDP(outBuf,nextNode)
		printErr(errWrite)

		if(incommingMessage == 9){
			listen.Close()
			break
		}
	}

}



func replicate(val int) {
	//print(val)
}
