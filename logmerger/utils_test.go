package logmerger

import (
	"fmt"
	"testing"
)

func TestClockConstruction(t *testing.T) {
	testClock := ConstructVclock([]string{"h1", "h2"}, []int{6, 4})
	want := "{\"h1\":6, \"h2\":4}"
	got := testClock.ReturnVCString()
	if want != got {
		t.Errorf("Clock Construction Errror\n want: %s\n got %s\n", want, got)
	}
}

func TestGetClockIds(t *testing.T) {
	testClock := ConstructVclock([]string{"h1", "h2"}, []int{6, 4})
	want := []string{"h1", "h2"}
	got := getClockIds(testClock)
	for i := range got {
		if want[i] != got[i] {
			t.Errorf("Clock Id extraction error\n want:%s\n got%s\n", want, got)
		}
	}
}

func Example_getClockIds() {
	exampleClock := ConstructVclock([]string{"Houston", "Apollo"}, []int{6, 4})
	clockIds := getClockIds(exampleClock)
	for _, host := range clockIds {
		fmt.Printf("id: %s, ", host)
	}
	// Output:
	// id: Houston, id: Apollo

}
