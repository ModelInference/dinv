// LogMerger
package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/bestchai/dinv/logmerger/vclock"
	//"reflect"
)

var usage = "logmerger log1.txt log2.txt"

const (
	INPUT_LOGS = 2
)

func main() {
	if len(os.Args) != INPUT_LOGS+1 {
		fmt.Printf("%s\n", usage)
		os.Exit(1)
	}
	for i := 1; i < len(os.Args); i++ {
		exists, err := fileExists(os.Args[i])
		if !exists {
			fmt.Printf("the file %s, does not exist\n%s\n", os.Args[1], err)
			os.Exit(1)
		}
	}
	//TODO refactor for n-logs later
	log1 := readLog(os.Args[1])
	log2 := readLog(os.Args[2])
	addNodeName("client.", log1)
	addNodeName("server.", log2)
	fmt.Println(log1[0])
	fmt.Println(log2[0])
	mergedLog := mergeLogs(log1, log2)
	writeLogToFile(mergedLog)

}

func addNodeName(name string, logs []Point) {
	for i := range logs {
		for j := range logs[i].Dump {
			//fmt.Println(dp.VarName)
			logs[i].Dump[j].VarName = name + logs[i].Dump[j].VarName
		}
	}
}

func writeLogToFile(log []Point) {

	file, _ := os.Create("daikonLog.dtrace")

	mapOfPoints := createMapOfLogsForEachPoint(log)
	writeDeclaration(file, mapOfPoints)
	writeValues(file, log)

}

func createMapOfLogsForEachPoint(log []Point) map[string][]Point {
	mapOfPoints := make(map[string][]Point, 0)

	for i := 0; i < len(log); i++ {
		mapOfPoints[log[i].LineNumber] = append(mapOfPoints[log[i].LineNumber], log[i])
	}

	return mapOfPoints
}

func writeDeclaration(file *os.File, mapOfPoints map[string][]Point) {

	file.WriteString("decl-version 2.0\n")
	file.WriteString("var-comparability none\n")
	file.WriteString("\n")
	for _, v := range mapOfPoints {
		point := v[0]
		file.WriteString(fmt.Sprintf("ppt p-%s:::%s\n", point.LineNumber, point.LineNumber))
		file.WriteString(fmt.Sprintf("ppt-type point\n"))

		for i := 0; i < len(point.Dump); i++ {
			file.WriteString(fmt.Sprintf("variable %s\n", point.Dump[i].VarName))
			file.WriteString(fmt.Sprintf("var-kind variable\n"))
			file.WriteString(fmt.Sprintf("dec-type %s\n", point.Dump[i].Type))
			file.WriteString(fmt.Sprintf("rep-type %s\n", point.Dump[i].Type))
			file.WriteString(fmt.Sprintf("comparability -1\n"))

		}
		file.WriteString("\n")

	}
}

func writeValues(file *os.File, log []Point) {
	for i := range log {
		point := log[i]
		file.WriteString(fmt.Sprintf("p-%s:::%s\n", point.LineNumber, point.LineNumber))
		file.WriteString(fmt.Sprintf("this_invocation_nonce\n"))
		file.WriteString(fmt.Sprintf("%d\n", i))
		for i := range point.Dump {
			variable := point.Dump[i]
			file.WriteString(fmt.Sprintf("%s\n", variable.VarName))
			if variable.Type == "int" {
				file.WriteString(fmt.Sprintf("%d\n", variable.Value))
			} else {
				file.WriteString(strings.Replace(fmt.Sprintf("%s", variable.Value), "\n", " ", -1) + "\n")
			}

			file.WriteString(fmt.Sprintf("1\n"))

		}
		file.WriteString("\n")

	}
}

func mergeLogs(log1, log2 []Point) []Point {

	mergedPoints := make([]Point, 0)
	for i := 0; i < len(log1); i++ {

		matchedPoints := findMatch(log1[i], log2)
		fmt.Println(matchedPoints)
		for j := 0; j < len(matchedPoints); j++ {
			mergedPoints = append(mergedPoints, mergePoints(matchedPoints[j], log1[i]))
		}
	}

	return mergedPoints

}

func mergePoints(p1, p2 Point) Point {
	var mergedPoint Point
	mergedPoint.Dump = append(p1.Dump, p2.Dump...)
	mergedPoint.LineNumber = p1.LineNumber + "-" + p2.LineNumber
	pVClock1, err := vclock.FromBytes(p1.VectorClock)
	printErr(err)
	pVClock2, err2 := vclock.FromBytes(p2.VectorClock)
	printErr(err2)
	temp := pVClock1.Copy()
	temp.Merge(pVClock2)
	mergedPoint.VectorClock = temp.Bytes()

	return mergedPoint

}

func findMatch(point Point, log []Point) []Point {
	matched := make([]Point, 0)
	pVClock, err := vclock.FromBytes(point.VectorClock)
	fmt.Println(pVClock)
	printErr(err)
	for i := 0; i < len(log); i++ {

		otherVClock, err2 := vclock.FromBytes(log[i].VectorClock)
		printErr(err2)
		if pVClock.Matches(otherVClock) {
			matched = append(matched, log[i])
		}
	}

	return matched
}

func readLog(filePath string) []Point {
	fileR, err := os.Open(filePath)
	printErr(err)

	fmt.Println("decoding " + filePath)
	decoder := gob.NewDecoder(fileR)

	pointArray := make([]Point, 0)

	var e error = nil
	for e == nil {
		var decodedPoint Point
		e = decoder.Decode(&decodedPoint)
		if e == nil {
			pointArray = append(pointArray, decodedPoint)
		} else {
			fmt.Println(e)
		}
	}

	return pointArray
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
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

func (nvp NameValuePair) String() string {
	return fmt.Sprintf("(%s,%s,%s)", nvp.VarName, nvp.Value, nvp.Type)
}

func (p Point) String() string {
	return fmt.Sprintf("%d : %s", p.LineNumber, p.Dump)
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
