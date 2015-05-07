package main

import (
	"fmt"
	"net"
	"time"

	"../govec"
	"encoding/gob"

	
	"os"
	"reflect"
	"strconv"
)

func main() {
 InstrumenterInit()

	Logger = govec.Initialize("Server", "testlog.log")
	conn, err := net.ListenPacket("udp", ":8080")
	//	if err != nil {
	//		fmt.Println(err)
	//		os.Exit(1)
	//	}
	printErr(err)

	for {
		if err != nil {
			printErr(err)
			continue
		}
		handleConn(conn)
		fmt.Println("some one connected!")
		vars28 := []interface{}{err,conn}
varsName28 := []string{"err","conn"}
point28 := createPoint(vars28, varsName28, 28)
encoder.Encode(point28)
	}
	conn.Close()

}

func handleConn(conn net.PacketConn) {
	var buf [512]byte

	_, addr, err := conn.ReadFrom(buf[0:])
	Logger.UnpackReceive("Received", buf[0:])
	printErr(err)
	msg := fmt.Sprintf("Hello There! time now is %s \n", time.Now().String())
	conn.WriteTo(Logger.PrepareSend("Sending", []byte(msg)), addr)
	vars42 := []interface{}{conn,Logger,err,msg,addr,buf}
varsName42 := []string{"conn","Logger","err","msg","addr","buf"}
point42 := createPoint(vars42, varsName42, 42)
encoder.Encode(point42)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

var Logger *govec.GoLog



var encoder *gob.Encoder

func InstrumenterInit() {
	fileW, _ := os.Create("../TestPrograms/serverUDP.go.txt")
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

