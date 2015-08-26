package instrumenter

import (
	"os"
	"testing"
)

func TestCopy(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("%s", err)
	}
	testdir := dir + "/test"
	os.MkdirAll(testdir, 0775)
	f, _ := os.Create(testdir + "/a.txt")
	f.WriteString("A is love")
	f, _ = os.Create(testdir + "/b.txt")
	f.WriteString("B is life")
	subTestDir := testdir + "/sub"
	os.MkdirAll(subTestDir, 0775)
	f, _ = os.Create(subTestDir + "/c.txt")
	f.WriteString("C was never here")
	err = InplaceDirectorySwap(testdir)
	os.RemoveAll(testdir)
	os.RemoveAll(testdir + "_orig")
	if err != nil {
		t.Error(err)
	}

}
