package chainkv

func foo(id, key int, last bool) bool {
	return (key%2 != id%2 && !last)
}
