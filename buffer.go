package arraybuffer

type ArrayBuffer struct {
	array     []byte
	chunkSize int
	count     int
	indices   chan int
}

func NewBuffer(chunkSize int, count int) *ArrayBuffer {
	ab := &ArrayBuffer{
		array:     make([]byte, chunkSize*count),
		chunkSize: chunkSize,
		count:     count,
		indices:   make(chan int, count),
	}

	for i := 0; i < count; i++ {
		ab.indices <- i
	}

	return ab
}

func (ab *ArrayBuffer) getBuffer(index int) []byte {
	return ab.array[index*ab.chunkSize : (index+1)*ab.chunkSize]
}

func (ab *ArrayBuffer) getIndex() int {
	return <-ab.indices
}

func (ab *ArrayBuffer) releaseIndex(index int) {
	for i := 0; i < ab.chunkSize; i++ {
		ab.array[index*ab.chunkSize+i] = 0
	}
	ab.indices <- index
}

type Array struct {
	parent  *ArrayBuffer
	indices []int
	length  int
}

func (ab *ArrayBuffer) NewArray() *Array {
	return &Array{
		parent:  ab,
		indices: make([]int, 0, 8),
		length:  0,
	}
}

func (a *Array) Append(data []byte) {
	for len(data) > 0 {
		curIdx := a.length / a.parent.chunkSize
		innerIdx := a.length % a.parent.chunkSize
		switch {
		case curIdx == len(a.indices):
			newIdx := a.parent.getIndex()
			a.indices = append(a.indices, newIdx)
			copyLen := len(data)
			if copyLen > a.parent.chunkSize {
				copyLen = a.parent.chunkSize
			}
			copy(a.parent.getBuffer(newIdx), data[:copyLen])
			data = data[copyLen:]
			a.length += copyLen
		default:
			copyLen := len(data)
			if copyLen > a.parent.chunkSize-innerIdx {
				copyLen = a.parent.chunkSize - innerIdx
			}
			copy(a.parent.getBuffer(a.indices[curIdx])[innerIdx:], data[:copyLen])
			data = data[copyLen:]
			a.length += copyLen

			innerIdx += copyLen
			if innerIdx == a.parent.chunkSize {
				curIdx++
				innerIdx = 0
			}
		}
	}
}

func (a *Array) Bytes() []byte {
	if len(a.indices) == 0 {
		return nil
	}

	result := make([]byte, a.length)
	curIdx := 0

	for _, idx := range a.indices[:len(a.indices)-1] {
		copy(result[curIdx:], a.parent.getBuffer(idx))
		curIdx += a.parent.chunkSize
	}

	copy(result[curIdx:], a.parent.getBuffer(a.indices[len(a.indices)-1])[:a.length%a.parent.chunkSize])

	return result
}

func (a *Array) Reset() {
	for _, idx := range a.indices {
		a.parent.releaseIndex(idx)
	}
	a.indices = a.indices[:0]
	a.length = 0
}
