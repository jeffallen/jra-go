package ebml

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

var BadFormat = errors.New("badly formatted Vint")

type Vint uint64

/*
type peekable struct {
	r	io.Reader
	peek	[]byte
	buf		[16]byte
}

func newPeekable(r io.Reader) (p *peekable) {
	p = &peekable{ r: r }
	p.peek = p.buf[0:0]
}

func (p *peekable)Read(buf []byte) (n int, err error) {
	if len(p.peek) > 0 {
		if len(p.peek) > len(buf) {
			n = copy(buf, p.peek[:len(buf)])
			p.peek = p.peek[len(buf):]
			return
		} else {
			n = copy(buf, p.peek)
			p.peek = p.buf[0:0]
			return
		}
		n, err = p.r.Read(buf)
	}
}

func (*peekable)ReadByte() (b byte, err error) {
	var buf [1]byte
	var n int

	n, err = r.Read(buf[:])
	if err != nil {
		return
	}

	if n == 1 {
		b = buf[0]
	} else {
		err = io.EOF
	}
	return
}
*/

func (m *Master)readVint() (x Vint, err error) {
	x = 0
	var b byte
	if b, err = m.r.ReadByte(); err != nil {
		return
	}
	suffix := byte(0)
	mask := byte(0x80)

	for suffix < 8 {
		if (b & mask) != 0 {
			x = Vint((uint64(b) & uint64(mask-1)) << uint64(8*suffix))
			if suffix != 0 {
				var mr uint64
				if mr, err = more(m.r, suffix); err != nil {
					return
				}
				x += Vint(mr)
			}
			return
		}
		suffix++
		mask = mask >> 1
	}

	err = BadFormat
	return
}

func more(r io.ByteReader, more byte) (x uint64, err error) {
	x = 0
	for more != 0 {
		var b byte
		if b, err = r.ReadByte(); err != nil {
			return
		}
		more--
		x += uint64(b) << (8 * more)
	}
	return
}

type element struct {
	id   Vint
	size Vint
	lr   *io.LimitedReader
}

func (m *Master)readElement() (e element, err error) {
	if e.id, err = m.readVint(); err != nil {
		return
	}
	if e.size, err = m.readVint(); err != nil {
		return
	}
	lr := io.LimitReader(m.r, int64(e.size))
	e.lr = lr.(*io.LimitedReader)
	return
}

func (e element)readInt() (x uint64, err error) {
	var buf [1]byte
	for i := e.lr.N; i > 0; i-- {
		var n int
		if n, err = e.lr.Read(buf[:]); err == nil {
			if n != 1 {
				err = errors.New("short read")
				return
			}
			x += uint64(buf[0]) << uint64(8*(i-1))
		} else {
			return
		}
	}
	return
}

type Unknown struct {
	Id	uint64
	Data []byte
}

type Master struct {
	r	*bufio.Reader
}

func NewMaster(r io.Reader) Master {
	return Master{ r: bufio.NewReader(r) }
}

func (m *Master)Next() (res interface {}, err error) {
	var e element
	if e, err = m.readElement(); err != nil {
		return
	}
	switch e.id {
	default:
		//data, err := ioutil.ReadAll(e.lr)
		if err == nil {
			res = Unknown{ Id: uint64(e.id), Data: nil }
		}
	case 0xA45DFA3:
		res, err = e.readHeader()
	case 0x8538067:
		res, err = e.readSegment()
	case 0x14D9B74:
		res, err = e.readMetaSeek()
	case 0xdbb:
		res, err = e.readSeek()
	}
	return
}

type Seek struct {
	SeekID []byte
	SeekPosition uint64
}

func (elt element)readSeek() (s Seek, err error) {
	s.SeekID, err = read

type MetaSeek struct {
	Seeks []Seek
}

func (elt element)readMetaSeek() (ms MetaSeek, err error) {
	m := NewMaster(elt.lr)
	ms.Seeks = make([]Seek, 0, 10)
	for err == nil {
		var x interface{}
		x, err = m.Next()
		if err == nil {
			switch y := x.(type) {
			case Unknown:
				fmt.Printf("unknown id: %x\n", y.Id)
			case Seek:
				ms.Seeks = append(ms.Seeks, y)
			}
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

type Segment struct {
	Master Master
}

func (elt element)readSegment() (s Segment, err error) {
	s.Master = NewMaster(elt.lr)
	return
}

type Header struct {
	Version            uint64
	ReadVersion        uint64
	MaxIdLength        uint64
	MaxSizeLength      uint64
	DocType            string
	DocTypeVersion     uint64
	DocTypeReadVersion uint64
}

func (elt element)readHeader() (h Header, err error) {
	// set to defaults
	h.Version = 1
	h.ReadVersion = 1
	h.MaxIdLength = 4
	h.MaxSizeLength = 8
	h.DocType = "matroska"
	h.DocTypeVersion = 1
	h.DocTypeReadVersion = 1

	m := NewMaster(elt.lr)

	for err == nil {
		var e element
		e, err = m.readElement()
		if err == nil {
			switch e.id {
			default:
				println("unknown id:", e.id)
			case 0x286:
				h.Version, err = e.readInt()
			case 0x2f7:
				h.ReadVersion, err = e.readInt()
			case 0x2f2:
				h.MaxIdLength, err = e.readInt()
			case 0x2f3:
				h.MaxSizeLength, err = e.readInt()
			case 0x287:
				h.DocTypeVersion, err = e.readInt()
			case 0x285:
				h.DocTypeReadVersion, err = e.readInt()
			case 0x282:
				var s []byte
				s, err = ioutil.ReadAll(e.lr)
				if err == nil {
					h.DocType = string(s)
				}
			}
		}
	}

	if err == io.EOF {
		// expected way to finish the loop, so clear the err for caller
		err = nil
	}

	return
}

// ex: ts=2
