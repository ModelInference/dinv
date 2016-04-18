package ricartagrawala

import (
	"net"
	"fmt"
	"bitbucket.org/bestchai/dinv/instrumenter"
)


type Controller struct {
	nodes map[int]*net.UDPAddr		//List of all nodes in the group
	id int
	hosts int						//number of hosts in the group
	listen *net.UDPConn				//Listening Port
}


func (c *Controller) SendCommand(msg string, host int) {
		m := Message{0,msg,id}
		out := instrumenter.Pack(m)
		//fmt.Printf("sending %s [ %d --> %d ] {body : %s}\n",msg,id, host,m.String())
		listen.WriteToUDP(out,nodes[host])
}

func NewController(id, hosts int) *Controller{
	c := new(Controller)
	c.id = id
	c.hosts = hosts
	lAddr, _ := net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT + id))
	c.listen, _ = net.ListenUDP("udp4",lAddr)
	c.listen.SetReadBuffer(1024)
	c.nodes = make(map[int]*net.UDPAddr)
	for i:=0; i<hosts; i++{
		//dont make a connection with yourself
		if i == id {
			continue
		} else {
			c.nodes[i], _ = net.ResolveUDPAddr("udp4", ":"+fmt.Sprintf("%d", BASEPORT + i))
		}
	}
	return c
}
