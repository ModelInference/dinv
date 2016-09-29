package main

import (
	"net"
)

func main() {
	capture(5)
	capture(test)

}

func test() {
	print("blegh")
}

func capture(i interface{}) {
	switch f := i.(type) {
	case int:
		print(f)
		break
	case func():
		f()
		break
	}

}
