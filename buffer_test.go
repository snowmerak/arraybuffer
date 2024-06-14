package arraybuffer_test

import (
	"fmt"
	"github.com/snowmerak/arraybuffer"
	"sync"
	"testing"
)

func TestArray_Append(t *testing.T) {
	ab := arraybuffer.NewBuffer(3, 30)
	a := ab.NewArray()

	a.Append([]byte("hello, "))
	a.Append([]byte("world! "))
	a.Append([]byte("start, "))
	a.Append([]byte("end! "))
	a.Append([]byte("goodbye!"))
	bs := a.Bytes()
	if string(bs) != "hello, world! start, end! goodbye!" {
		t.Errorf("unexpected result: %s", string(bs))
	}

	a.Reset()
}

func TestArray_Append2(t *testing.T) {
	ab := arraybuffer.NewBuffer(4096, 1048572)
	a := ab.NewArray()

	wg := sync.WaitGroup{}
	for i := 0; i < 4096; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bs := make([]byte, 4096)
			for j := 0; j < 4096; j++ {
				bs[j] = byte(i)
			}
			for j := 0; j < 255; j++ {
				a.Append(bs)
			}
			b := a.Bytes()
			fmt.Printf("len: %d\n", len(b))
			a.Reset()
		}(i)
	}

	wg.Wait()
}
