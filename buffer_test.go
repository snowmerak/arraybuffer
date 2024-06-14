package arraybuffer_test

import (
	"github.com/snowmerak/arraybuffer"
	"sync"
	"testing"
)

func TestArray_Append(t *testing.T) {
	ab := arraybuffer.New(3, 30)
	a := ab.List()

	a.Write([]byte("hello, "))
	a.Write([]byte("world! "))
	a.Write([]byte("start, "))
	a.Write([]byte("end! "))
	a.Write([]byte("goodbye!"))
	bs := a.Bytes()
	if string(bs) != "hello, world! start, end! goodbye!" {
		t.Errorf("unexpected result: %s", string(bs))
	}

	a.Reset()
}

func TestArray_Append2(t *testing.T) {
	ab := arraybuffer.New(4096, 1048572)

	wg := sync.WaitGroup{}
	for i := 0; i < 1024; i++ {
		a := ab.List()
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bs := make([]byte, 4)
			for j := 0; j < 4; j++ {
				bs[j] = byte(i)
			}
			for j := 0; j < 1024; j++ {
				a.Write(bs)
			}
			b := a.Bytes()
			if len(b) != 4096 {
				t.Errorf("unexpected length: %d", len(b))
			}
			a.Reset()
		}(i)
	}

	wg.Wait()
}
