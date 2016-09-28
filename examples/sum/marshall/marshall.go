package marshall

const IntSize = 4

func MarshallInts(args []int) []byte {
	var i, j uint
	marshalled := make([]byte, len(args)*IntSize, len(args)*IntSize)
	for j = 0; int(j) < len(args); j++ {
		for i = 0; i < IntSize; i++ {
			marshalled[(j*IntSize)+i] = byte(args[j] >> ((IntSize - 1 - i) * 8))
		}
	}
	//@dump
	return marshalled
}

func UnmarshallInts(args []byte) []int {
	var i, j uint
	unmarshalled := make([]int, len(args)/IntSize, len(args)/IntSize)
	for j = 0; int(j) < len(args)/IntSize; j++ {
		for i = 0; i < IntSize; i++ {
			unmarshalled[j] += int(args[IntSize*(j+1)-1-i] << (i * 8))
		}
	}
	return unmarshalled
}
