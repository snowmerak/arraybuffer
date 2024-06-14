package arraybuffer

import (
	"errors"
	"fmt"
	"io"
	"time"
)

type ArrayBuffer struct {
	array     []byte
	chunkSize int
	count     int
	fragments chan *fragment
}

func New(chunkSize int, count int) *ArrayBuffer {
	ab := &ArrayBuffer{
		array:     make([]byte, chunkSize*count),
		chunkSize: chunkSize,
		count:     count,
		fragments: make(chan *fragment, count),
	}

	for i := 0; i < count; i++ {
		ab.fragments <- &fragment{
			buffer:     ab,
			startIndex: i * chunkSize,
			endIndex:   (i + 1) * chunkSize,
			length:     0,
		}
	}

	return ab
}

func (ab *ArrayBuffer) getFragment() (*fragment, bool) {
	retries := 0
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	for retries < 5 {
		select {
		case v, ok := <-ab.fragments:
			if ok {
				return v, true
			}
		case <-timer.C:
			retries++
			timer.Reset(time.Millisecond)
		}
	}
	return nil, false
}

func (ab *ArrayBuffer) releaseIndex(f *fragment) {
	ab.fragments <- f
}

type fragment struct {
	buffer     *ArrayBuffer
	startIndex int
	endIndex   int
	readIdx    int
	length     int
}

func (f *fragment) bytes() []byte {
	return f.buffer.array[f.startIndex+f.readIdx : f.startIndex+f.length]
}

func (f *fragment) reset() {
	for i := f.startIndex; i <= f.endIndex; i++ {
		f.buffer.array[i] = 0
	}
	f.length = 0

	f.buffer.releaseIndex(f)
}

func (f *fragment) write(bs []byte) (int, error) {
	if f.length >= f.buffer.chunkSize {
		return 0, nil
	}

	copyLen := len(bs)
	if f.length+copyLen > f.buffer.chunkSize {
		copyLen = f.buffer.chunkSize - f.length
	}

	copy(f.buffer.array[f.startIndex+f.length:], bs[:copyLen])

	f.length += copyLen
	return copyLen, nil
}

func (f *fragment) read(bs []byte) (int, error) {
	if f.readIdx >= f.length {
		return 0, nil
	}

	copyLen := len(bs)
	if f.readIdx+copyLen > f.length {
		copyLen = f.length - f.readIdx
	}

	copy(bs, f.buffer.array[f.startIndex+f.readIdx:f.startIndex+f.readIdx+copyLen])
	f.readIdx += copyLen
	return copyLen, nil
}

type List struct {
	buffer    *ArrayBuffer
	fragments []*fragment
	length    int
	readIdx   int
}

func (l *List) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		l.readIdx = int(offset)
	case io.SeekCurrent:
		readIdx := l.readIdx
		readIdx += int(offset) / l.buffer.chunkSize
		if readIdx < 0 || readIdx >= len(l.fragments) {
			return 0, errors.New(fmt.Sprintf("out of range: %d / %d", readIdx, len(l.fragments)))
		}
		fragmentReadIdx := int(offset) % l.buffer.chunkSize
		if fragmentReadIdx < 0 || fragmentReadIdx >= l.fragments[readIdx].length {
			return 0, errors.New(fmt.Sprintf("out of range: %d / %d", fragmentReadIdx, l.fragments[readIdx].length))
		}
		l.readIdx = readIdx
		l.fragments[l.readIdx].readIdx = fragmentReadIdx
	case io.SeekEnd:
		if l.length > -int(offset) || offset > 0 {
			return 0, errors.New(fmt.Sprintf("out of range: %d / %d", offset, l.length))
		}
		offset = int64(l.length) + offset
		l.readIdx = int(offset) / l.buffer.chunkSize
		l.fragments[l.readIdx].readIdx = int(offset) % l.buffer.chunkSize
	}

	return int64(l.readIdx), nil
}

func (l *List) Close() error {
	l.Reset()
	return nil
}

func (ab *ArrayBuffer) List() *List {
	return &List{
		buffer: ab,
	}
}

func (l *List) Write(bs []byte) (int, error) {
	written := 0

	if len(l.fragments) < 1 {
		v, ok := l.buffer.getFragment()
		if !ok {
			return 0, errors.New("no fragment available")
		}

		l.fragments = append(l.fragments, v)
	}

	lastFragment := l.fragments[len(l.fragments)-1]
	for len(bs) > 0 {
		if lastFragment.length == l.buffer.chunkSize {
			v, ok := l.buffer.getFragment()
			if !ok {
				return written, errors.New("no fragment available")
			}

			l.fragments = append(l.fragments, v)
			lastFragment = l.fragments[len(l.fragments)-1]
		}

		n, err := lastFragment.write(bs)
		if err != nil {
			return written, fmt.Errorf("fragment write error: %w", err)
		}

		written += n

		l.length += n
		bs = bs[n:]
	}

	return written, nil
}

func (l *List) Read(bs []byte) (int, error) {
	read := 0
	for len(bs) > 0 {
		if len(l.fragments) < 1 || l.readIdx >= len(l.fragments) {
			break
		}

		n, err := l.fragments[l.readIdx].read(bs)
		if err != nil {
			return read, fmt.Errorf("fragment read error: %w", err)
		}

		read += n
		bs = bs[n:]

		if l.fragments[l.readIdx].readIdx >= l.fragments[l.readIdx].length {
			l.readIdx++
		}
	}

	return read, nil
}

func (l *List) Bytes() []byte {
	bs := make([]byte, l.length)
	offset := 0
	for _, f := range l.fragments {
		copy(bs[offset:], f.bytes())
		offset += f.length
	}
	return bs
}

func (l *List) Reset() {
	for _, f := range l.fragments {
		f.reset()
	}
	l.fragments = l.fragments[:0]
	l.length = 0
}
