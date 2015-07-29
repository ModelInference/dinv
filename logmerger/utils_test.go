package logmerger

import (
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
