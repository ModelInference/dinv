package main

import (
	"fmt"
	"github.com/arcaneiceman/GoVector/capture"
	"bitbucket.org/bestchai/dinv/dinvRT"
	"net"
	"os"
)

//var debug = false

//dump
func main() {
	conn, err := net.ListenPacket("udp", ":9090")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	PrintErr(err)

	//main loop
	for {
		if err != nil {
			PrintErr(err)
			continue
		}
		handleConn(conn)
		//fmt.Println("some one connected!")
	}
	conn.Close()
}

func handleConn(conn net.PacketConn) {
	var buf [1024]byte
	var term1, term2, coeff, lin int

	_, addr, err := capture.ReadFrom(conn.ReadFrom,buf[0:])

	dinvRT.Dump("main_linn_39_SIZEOFINT,main_linn_39_conn,main_linn_39_buf,main_linn_39_term1,main_linn_39_term2,main_linn_39_coeff,main_linn_39_lin,main_linn_39_addr,main_linn_39_err", SIZEOFINT, conn, buf, term1, term2, coeff, lin, addr, err)
	dinvRT.Track("main_linn_40_SIZEOFINT,main_linn_40_conn,main_linn_40_buf,main_linn_40_term1,main_linn_40_term2,main_linn_40_coeff,main_linn_40_lin,main_linn_40_addr,main_linn_40_err", SIZEOFINT, conn, buf, term1, term2, coeff, lin, addr, err)
	PrintErr(err)

	uArgs := UnmarshallInts(buf)
	term1, term2, coeff = uArgs[0], uArgs[1], uArgs[2]
	lin = coeff*term1 + term2
	//if debug {
	//	fmt.Printf("C: %d*%d + %d = %d\n", coeff, term1, term2, lin)
	//}
	msg := MarshallInts([]int{lin})

	dinvRT.Dump("main_linn_51_SIZEOFINT,main_linn_51_conn,main_linn_51_buf,main_linn_51_term1,main_linn_51_term2,main_linn_51_coeff,main_linn_51_lin,main_linn_51_addr,main_linn_51_err,main_linn_51_uArgs,main_linn_51_msg", SIZEOFINT, conn, buf, term1, term2, coeff, lin, addr, err, uArgs, msg)
	capture.WriteTo(conn.WriteTo,msg,addr)
}

const (
	SIZEOFINT = 4
)

func PrintErr(err error) {
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
	return marshalled
}

func UnmarshallInts(args [1024]byte) []int {
	var i, j uint
	unmarshalled := make([]int, len(args)/SIZEOFINT, len(args)/SIZEOFINT)
	for j = 0; int(j) < len(args)/SIZEOFINT; j++ {
		for i = 0; i < SIZEOFINT; i++ {
			unmarshalled[j] += int(args[SIZEOFINT*(j+1)-1-i] << (i * 8))
		}
	}
	return unmarshalled
}
