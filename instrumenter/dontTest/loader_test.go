package instrumenter

import "testing"

func TestLoad(t *testing.T) {
	p := getWrappers2("/home/stewartgrant/go/src/bitbucket.org/bestchai/dinv/TestPrograms/t2/client/breakup", "client")
	if p == nil {
		t.Error("Fail cannot load package client")
	}
}
