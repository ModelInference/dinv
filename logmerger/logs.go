//logs contains function for working with and constructing log files.
//The log files include, govec, points, and trace.

package logmerger

//readLog attempts to extract an array of program points from a log
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/arcaneiceman/GoVector/govec/vclock"
)

//file. If the log file does not exist or is unreadable, readLog
//panics. Otherwise an array of program points is returned
func readLog(filePath string) []Point {
	fileR, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(fileR)
	pointArray := make([]Point, 0)
	var e error = nil
	for e == nil {
		var decodedPoint Point
		e = decoder.Decode(&decodedPoint)
		fixJsonEncodingTypeConversion(&decodedPoint)
		if e == nil {
			//logger.Printf(decodedPoint.String())
			pointArray = append(pointArray, decodedPoint)
		}
	}
	return pointArray
}

//Json encoding is fast, but it can mess with the types of the
//variables passed to it. For instance integers are converted to
//floating points by adding .00 to them. This function corrects for
//these mistakes and returns the points to their origianl state.
func fixJsonEncodingTypeConversion(point *Point) {
	for i := range point.Dump {
		if point.Dump[i].Type == "int" {
			point.Dump[i].Value = int(point.Dump[i].Value.(float64))
			// fmt.Printf("type :%s\t value: %s\n",reflect.TypeOf(point.Dump[i].Value).String(),point.Dump[i].value())
		}
	}
}

//Inject missing points ensures that the log of points contains
//incremental vector clocks.
func injectMissingPoints(points []Point, log *golog) []Point {
	pointIndex, goLogIndex := 0, 0
	injectedPoints := make([]Point, 0)
	//itterate over all the point logs
	indexFound := false
	for pointIndex < len(points) && goLogIndex < len(log.clocks) {
		//setup for a do while loop
		pointClock, _ := vclock.FromBytes(points[pointIndex].VectorClock)
		ticks, found := pointClock.FindTicks(log.id)
		//The point log contains the incremental index, append the
		//point to the log
		if found && int(ticks) == (goLogIndex+1) {
			injectedPoints = append(injectedPoints, points[pointIndex])
			//fmt.Printf("Appending clock %d aboslute %d\n", pointIndex, goLogIndex)
			pointIndex++
			indexFound = true
			//The point log contained the index and has advanced to the
			//next
		} else if found && int(ticks) != (goLogIndex+1) && indexFound {
			goLogIndex++
			//fmt.Printf("Advancing log by one step index %d\n", goLogIndex)
			indexFound = false
			//The point log did not contain the index, inject a
			//supplementary one
		} else if goLogIndex < len(log.clocks) {
			//fmt.Printf("Injecting Clock %s into log %s\n", log.clocks[goLogIndex].ReturnVCString(), log.id)
			newPoint := new(Point)
			//newPoint.Id = points[0].Id //this may be bad (attempt to fix output logs) BUG

			newPoint.VectorClock = log.clocks[goLogIndex].Bytes()
			injectedPoints = append(injectedPoints, *newPoint)
			goLogIndex++
			indexFound = false
		}
	}
	//The golog may contain a set of points never loged by the the
	//points, inject all of them
	for goLogIndex < len(log.clocks) {
		//fmt.Printf("Injecting Clock %s into log %s\n", log.clocks[goLogIndex].ReturnVCString(), log.id)
		newPoint := new(Point)
		newPoint.VectorClock = log.clocks[goLogIndex].Bytes()
		injectedPoints = append(injectedPoints, *newPoint)
		goLogIndex++
	}
	return injectedPoints
}

//addBaseLog Injects a single valued vector clock as the base entry of
//a log. The base clock acts as a uniform starting point for
//computations being done to the logs.
func addBaseLog(name string, log []Point) []Point {
	clock := vclock.New()
	clock.Tick(name)
	first := new(Point)
	first.VectorClock = clock.Bytes()
	baseLog := make([]Point, 0)
	baseLog = append(baseLog, *first)
	for i := range log {
		baseLog = append(baseLog, log[i])
	}
	return baseLog

}

//Host Renaming structures
func replaceIds(pointLog [][]Point, goLogs []*golog, scheme string) {
	_, ok := namingSchemes[scheme]
	idMap := make(map[string]string)
	if !ok {
		if scheme != "" {
			fmt.Printf("Warning: unknown id naming scheme \"%s\", id's unchanged\n", scheme)
		}
		scheme = "default"
		defaults := make([]string, 0)
		for i := range goLogs {
			defaults = append(defaults, goLogs[i].id)
		}
		namingSchemes[scheme] = defaults
	}
	for i := range goLogs {
		idMap[goLogs[i].id] = namingSchemes[scheme][i]
	}
	for i := range pointLog {
		replace := regexp.MustCompile(goLogs[i].id)
		for j := range pointLog[i] {
			pointLog[i][j].Id = replace.ReplaceAllString(pointLog[i][j].Id, idMap[goLogs[i].id])
			for k := range pointLog[i][j].Dump {
				pointLog[i][j].Dump[k].VarName = idMap[goLogs[i].id] + "-" + pointLog[i][j].Dump[k].VarName
			}
			oldClock, _ := vclock.FromBytes(pointLog[i][j].VectorClock)
			newClock := swapClockIds(oldClock, idMap)
			pointLog[i][j].VectorClock = newClock.Bytes()
		}
		for j := range goLogs[i].clocks {
			goLogs[i].clocks[j] = swapClockIds(goLogs[i].clocks[j], idMap)
			goLogs[i].messages[j] = replace.ReplaceAllString(goLogs[i].messages[j], idMap[goLogs[i].id])
		}
		goLogs[i].id = idMap[goLogs[i].id]
	}
}

/* reading govec logs */
func ParseGologFile(filename string) (*golog, error) {
	var govecRegex string = "(\\S*) ({.*})\n(.*)"
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var text string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text += "\n" + scanner.Text()
	}
	log, err := GoLogFromString(text, govecRegex)
	if err != nil {
		defer file.Close()
		return nil, err
	}
	return log, nil
}

func GoLogFromString(clockLog, regex string) (*golog, error) {
	rex := regexp.MustCompile(regex)
	matches := rex.FindAllStringSubmatch(clockLog, -1)
	if len(matches) <= 0 {
		return nil, fmt.Errorf("No matches found")
	}
	id := matches[0][1]
	messages := make([]string, 0)
	rawClocks := make([]string, 0)
	for i := range matches {
		rawClocks = append(rawClocks, matches[i][2])
		messages = append(messages, matches[i][3])
	}

	vclocks := make([]vclock.VClock, 0)
	for i := range rawClocks {
		clock, err := ClockFromString(rawClocks[i], "\"([A-Za-z0-9_]+)\":([0-9]+)")
		if clock == nil || err != nil {
			return nil, err
		}
		vclocks = append(vclocks, clock)
	}
	log := &golog{id, vclocks, messages}
	return log, nil
}

func swapClockIds(oldClock vclock.VClock, idMap map[string]string) vclock.VClock {
	ids := make([]string, 0)
	ticks := make([]int, 0)
	for id := range idMap {
		tick, _ := oldClock.FindTicks(id)
		if tick > 0 {
			ids = append(ids, idMap[id])
			ticks = append(ticks, int(tick))
		}
	}
	return ConstructVclock(ids, ticks)
}

//writeLogToFile produces a daikon dtrace file based on a log
//represented as an array of points
func writeLogToFile(log []Point, filename string) {
	if len(filename) > 50 {
		filename = Hash(filename)
	}
	filenameWithExtenstion := fmt.Sprintf("%s.dtrace", filename)
	file, err := os.Create(filenameWithExtenstion)
	if err != nil {
		logger.Panic(err)
	}
	mapOfPoints := createMapOfLogsForEachPoint(log)
	writeDeclaration(file, mapOfPoints)
	writeValues(file, log)
}

//createMapOfLogsForEachPoint buckets points based on the line number
//they occur on. The map corresponding to each unique line number is
//returned
func createMapOfLogsForEachPoint(log []Point) map[string][]Point {
	mapOfPoints := make(map[string][]Point, 0)
	for i := 0; i < len(log); i++ {
		mapOfPoints[log[i].Id] = append(mapOfPoints[log[i].Id], log[i])
	}
	return mapOfPoints
}

//writeDeclaration writes out variable names and their types to the
//specified open file. The declarations are in a Daikon readable
//format
func writeDeclaration(file *os.File, mapOfPoints map[string][]Point) {
	file.WriteString("decl-version 2.0\n")
	file.WriteString("var-comparability none\n")
	file.WriteString("\n")
	for _, v := range mapOfPoints {
		point := v[0]
		file.WriteString(fmt.Sprintf("ppt p-%s:::%s\n", point.Id, point.Id))
		file.WriteString(fmt.Sprintf("ppt-type point\n"))
		for i := 0; i < len(point.Dump); i++ {
			//TODO work with types we cant handle
			if point.Dump[i].Type == "" {
				continue
			}
			file.WriteString(fmt.Sprintf("variable %s\n", point.Dump[i].VarName))
			file.WriteString(fmt.Sprintf("var-kind variable\n"))
			file.WriteString(fmt.Sprintf("dec-type %s\n", point.Dump[i].Type))
			file.WriteString(fmt.Sprintf("rep-type %s\n", point.Dump[i].Type))
			file.WriteString(fmt.Sprintf("comparability -1\n"))
		}
		file.WriteString("\n")

	}
}

//writeValeus outputs variable values and their associated line
//numbers. The output is in a Daikon readable format.
func writeValues(file *os.File, log []Point) {
	for i := range log {
		point := log[i]
		file.WriteString(fmt.Sprintf("p-%s:::%s\n", point.Id, point.Id))
		file.WriteString(fmt.Sprintf("this_invocation_nonce\n"))
		file.WriteString(fmt.Sprintf("%d\n", i))
		for i := range point.Dump {
			//TODO work with types we cant handle
			variable := point.Dump[i]
			if variable.Type == "" {
				continue
			}
			file.WriteString(fmt.Sprintf("%s\n", variable.VarName))

			file.WriteString(fmt.Sprintf("%s\n", variable.value()))
			file.WriteString(fmt.Sprintf("1\n"))
		}
		file.WriteString("\n")

	}
}

//Name value pair matches variable names to their values, along with
//their type
type NameValuePair struct {
	VarName string
	Value   interface{}
	Type    string
}

//String representation of a name value pair
func (nvp NameValuePair) String() string {
	return fmt.Sprintf("%s=%s , ", nvp.VarName, nvp.value())
}

//returns the value of the Name value pair as a string
//TODO catch and print all possible reflected types
func (nvp NameValuePair) value() string {
	v := reflect.ValueOf(nvp.Value)
	switch v.Kind() {
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", v.Float())
	case reflect.String:
		return fmt.Sprintf("\"%s\"", strings.Replace(fmt.Sprintf("%s", v.String()), "\n", " ", -1))
	default:
		return ""
	}
}

//Point is a representation of a program point. Name value pair is the
//variable values at that program point. LineNumber is the line the
//variables were gathered on. VectorClock is byte valued vector clock
//at the time the program point was logged
type Point struct {
	Dump               []NameValuePair
	Id                 string
	VectorClock        []byte
	CommunicationDelta int
}

type PointLogs map[string]map[uint64]Point

//String representation of a program point
func (p Point) String() string {
	dumpstring := ""
	for _, dump := range p.Dump {
		dumpstring = dumpstring + dump.String()
	}
	clock, _ := vclock.FromBytes(p.VectorClock)
	return fmt.Sprintf("%s { %s } { %s }", p.Id, dumpstring, clock.ReturnVCString())
}

type ById []Point

func (p ById) Len() int           { return len(p) }
func (p ById) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ById) Less(i, j int) bool { return p[i].Id < p[j].Id }

type golog struct {
	id       string
	clocks   []vclock.VClock
	messages []string
}

func (g *golog) String() string {
	var text string
	for i := range g.clocks {
		text += fmt.Sprintf("id: %s\t clock: %s\n %s\n", g.id, g.clocks[i].ReturnVCString(), g.messages[i])
	}
	return text
}

func (g *golog) Len() int {
	return len(g.clocks)
}

func (g *golog) Less(i, j int) bool {
	iclock, jclock := g.clocks[i], g.clocks[j]
	iticks, _ := iclock.FindTicks(g.id)
	jticks, _ := jclock.FindTicks(g.id)
	if iticks < jticks {
		return true
	}
	return false
}

func (g *golog) Swap(i, j int) {
	tmp := g.clocks[i]
	g.clocks[i] = g.clocks[j]
	g.clocks[j] = tmp
}

var namingSchemes = map[string][]string{
	"colors": []string{
		"blue", "red", "green", "purple", "black", "orange", "yellow", "gold", "white", "pink", "azure", "brown", "cobalt", "cyan", "grey", "indigo", "jade"},
	"fruits":       []string{"Apple", "Banana", "Apricot", "Strawberry", "Orange", "Grape", "Raspberry", "Blackberry", "Blueberry", "WaterMelon", "Rambutan", "Lanzones", "Pears", "Plums", "Peaches", "Pineapple", "Cantaloupe", "Papaya", "Jackfruit", "Durian"},
	"philosophers": []string{"Aristotle", "Chomsky", "Locke", "Nietzsche", "Plato"},
}

//"philosophers": []string{"Abelard", "Adorno", "Aquinas", "Arendt", "Aristotle", "Augustine", "Bacon", "Barthes", "Bataille", "Baudrillard", "Beauvoir", "Benjamin", "Berkeley", "Butler", "Camus", "Chomsky", "Cixous", "Deleuze", "Derrida", "Descartes", "Dewey", "Foucault", "Gadamer", "Habermas", "Haraway", "Hegel", "Heidegger", "Hobbes", "Hume", "Husserl", "Irigaray", "James", "Immanuel", "Kristeva", "Tzu", "Levinas", "Locke", "Lyotard", "Merleau-Ponty", "Mill", "Moore", "Nietzsche", "Plato", "Quine", "Rand", "Rousseau", "Sartre", "Schopenhauer", "Spinoza", "Wittgenstein"},
