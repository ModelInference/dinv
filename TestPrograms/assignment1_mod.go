
package main

import (
	"../govec"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
)

func main() {
 InstrumenterInit()

	Logger = govec.Initialize("Client", "testclient.log")

	rAddr, errR := net.ResolveUDPAddr("udp4", ":8080")
	printErr(errR)
	lAddr, errL := net.ResolveUDPAddr("udp4", ":18585")
	printErr(errL)

	conn, errDial := net.DialUDP("udp", lAddr, rAddr)
	printErr(errDial)

	// sending UDP packet to specified address and port
	msg := "get me the message !"
	_, errWrite := conn.Write(Logger.PrepareSend("Asking time", []byte(msg)))
	printErr(errWrite)

	// Reading the response message
	var buf [1024]byte
	n, errRead := conn.Read(buf[0:])
	printErr(errRead)
	incoming_msg := string(Logger.UnpackReceive("Received", buf[:n]))
	fmt.Println(">>>" + incoming_msg)
	vars32 := []interface{}{n,errR,errL,errDial,msg,conn,incoming_msg,lAddr,rAddr,Logger,errWrite,errRead}
varsName32 := []string{"n","errR","errL","errDial","msg","conn","incoming_msg","lAddr","rAddr","Logger","errWrite","errRead"}
point32 := createPoint(vars32, varsName32, 32)
encoder.Encode(point32)

	os.Exit(0)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var Logger *govec.GoLog



var encoder *gob.Encoder

func InstrumenterInit() {
	fileW, _ := os.Create("../TestPrograms/assignment1.go.txt")
	encoder = gob.NewEncoder(fileW)
}

func createPoint(vars []interface{}, varNames []string, lineNumber int) Point {

	length := len(varNames)
	dumps := make([]NameValuePair, 0)
	for i := 0; i < length; i++ {

		if vars[i] != nil && ((reflect.TypeOf(vars[i]).Kind() == reflect.String) || (reflect.TypeOf(vars[i]).Kind() == reflect.Int)) {
			var dump NameValuePair
			dump.VarName = varNames[i]
			dump.Value = vars[i]
			dump.Type = reflect.TypeOf(vars[i]).String()
			dumps = append(dumps, dump)
		}
	}

	point := Point{dumps, strconv.Itoa(lineNumber), Logger.GetCurrentVC()}
	return point
}

type Point struct {
	Dump        []NameValuePair
	LineNumber  string
	VectorClock []byte
}

type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}

//func (nvp NameValuePair) String() string {
//	return fmt.Sprintf("(%!s(MISSING),%!s(MISSING),%!s(MISSING))", nvp.VarName, nvp.Value, nvp.Type)
//}

//func (p Point) String() string {
//	return fmt.Sprintf("%!s(MISSING) : %!s(MISSING)", p.LineNumber, p.Dump)
//}

