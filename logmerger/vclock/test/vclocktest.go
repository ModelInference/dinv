// vclocktest
package main

import (
	"./vclock"
	"fmt"
)

// func main() {
// 	vc1 := vclock.New()
// 	vc1.Update("A", 1)

// 	vc2 := vc1.Copy()
// 	vc2.Update("B", 0)
// 	vc1.PrintVC()
// 	vc2.PrintVC()
// 	fmt.Println(vc2.Matches(vc1)) // true
// 	fmt.Println(vc1.Matches(vc2)) // true

// 	vc1.Update("C", 5)
// 	vc1.PrintVC()
// 	vc2.PrintVC()
// 	fmt.Println(vc1.Matches(vc2)) // false
// 	fmt.Println(vc1.Matches(vc2)) // true

// 	vc2.Merge(vc1)

// 	vc1.PrintVC()
// 	vc2.PrintVC()
// 	fmt.Println(vc1.Matches(vc2)) // true

// 	data := vc2.Bytes()
// 	fmt.Printf("%#v\n", string(data))

// 	vc3, err := vclock.FromBytes(data)
// 	if err != nil {
// 		panic(err)
// 	}
// 	vc2.PrintVC()
// 	vc3.PrintVC()
// 	fmt.Println(vc3.Matches(vc2)) // true
// }
