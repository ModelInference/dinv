package main

/* client benchmarks takes the following arguments */
// [1] Num Ints
// [2] Num Floats
// [3] Num Bool
// [4] num bytes
// [5] size bytes
// [6] number to execute

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"bitbucket.org/bestchai/dinv/dinvRT"
)

var (
	Ints      int
	Floats    int
	Bools     int
	Bytes     int
	BytesSize int
	Execute   int
)

func main() {
	Ints = ArgToInt(1)
	Floats = ArgToInt(2)
	Bools = ArgToInt(3)
	Bytes = ArgToInt(4)
	BytesSize = ArgToInt(5)
	Execute = ArgToInt(6)

	id := ""
	intid := ""
	floatid := ""
	boolid := ""
	byteid := ""

	IntVals := make([]int, Ints)
	for i := 0; i < Ints; i++ {
		IntVals[i] = i
		intid += fmt.Sprintf("int-%d,", i)
		id += fmt.Sprintf("int-%d,", i)
	}
	intid = intid[0 : len(intid)-1]

	FloatVals := make([]float32, Floats)
	for i := 0; i < Floats; i++ {
		FloatVals[i] = float32(i)
		floatid += fmt.Sprintf("float-%d,", i)
		id += fmt.Sprintf("float-%d,", i)
	}
	floatid = floatid[0 : len(floatid)-1]

	BoolVals := make([]bool, Bools)
	for i := 0; i < Bools; i++ {
		BoolVals[i] = true
		boolid += fmt.Sprintf("bool-%d,", i)
		id += fmt.Sprintf("bool-%d,", i)
	}
	boolid = boolid[0 : len(boolid)-1]

	ByteVals := make([][]byte, Bytes)
	for i := 0; i < Bytes; i++ {
		ByteVals[i] = make([]byte, BytesSize)
		byteid += fmt.Sprintf("bytes-%d,", i)
		id += fmt.Sprintf("bytes-%d,", i)
	}
	byteid = byteid[0 : len(byteid)-1]
	id = id[0 : len(id)-1]

	Aggregate := make([]interface{}, 0)
	for i := range IntVals {
		Aggregate = append(Aggregate, IntVals[i])
	}
	for i := range FloatVals {
		Aggregate = append(Aggregate, FloatVals[i])
	}
	for i := range BoolVals {
		Aggregate = append(Aggregate, BoolVals[i])
	}
	for i := range ByteVals {
		Aggregate = append(Aggregate, ByteVals[i])
	}
	//Run the test
	start := time.Now()
	for i := 0; i < Execute; i++ {
		dinvRT.Track("", id, Aggregate...)
		dinvRT.Pack(nil)
	}
	fmt.Println(time.Since(start).Nanoseconds())

}

func ArgToInt(index int) int {
	v, err := strconv.Atoi(os.Args[index])
	if err != nil {
		panic(err)
	}
	return v
}
