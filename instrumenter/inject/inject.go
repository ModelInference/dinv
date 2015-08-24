package inject

import (
	"encoding/gob"
	"fmt"
	"os"
	"reflect"

	"bitbucket.org/bestchai/dinv/instrumenter"

	"github.com/arcaneiceman/GoVector/govec"
	"github.com/arcaneiceman/GoVector/govec/vclock"
)

//TODO move structs to seperate file remove duplication in log merger

/* Injection Code */
//injection code is used to dynamicly write an injection file,
//containing methods called by dump statements

//header_code contains all the needed imports for the injection code,
//and is designed to have the package name written at runtime

var Encoder *gob.Encoder //global
var packageName string

//body code contains utility functions called by the code injected at
//dump statements
//TODO add comments to the inject code
//TODO build array of acceptable types for encoding
//TODO make the logger an argument to CreatePoint

func InstrumenterInit(pname string) {
	if Encoder == nil {
		packageName = pname
		id := instrumenter.GetId()
		encodedLogname := fmt.Sprintf("%s-%sEncoded.txt", packageName, id)
		encodedLog, _ := os.Create(encodedLogname)
		Encoder = gob.NewEncoder(encodedLog)
	}
}

func CreatePoint(vars []interface{}, varNames []string, id string, logger *govec.GoLog, hash string) Point {
	numVars := len(varNames)
	dumps := make([]NameValuePair, 0)
	hashedId := hash + "_" + id
	for i := 0; i < numVars; i++ {
		if vars[i] != nil {
			switch reflect.TypeOf(vars[i]).Kind() {
			case reflect.String, reflect.Int:
				var dump NameValuePair
				dump.VarName = varNames[i]
				dump.Value = vars[i]
				dump.Type = reflect.TypeOf(vars[i]).String()
				dumps = append(dumps, dump)
			}
		}
	}
	point := Point{dumps, hashedId, logger.GetCurrentVC()}
	return point
}

func Local(logger *govec.GoLog, id string) {
	logger.LogLocalEvent(fmt.Sprintf("Dump @ id %s", id))
}

type Point struct {
	Dump        []NameValuePair
	Id          string
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
	clock, _ := vclock.FromBytes(p.VectorClock)
	return fmt.Sprintf("%s\n%s %s\nVClock : %s\n\n", p.Id, clock.ReturnVCString())
}
