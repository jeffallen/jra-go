package stl

import (
	"bytes"
	"encoding/binary"
	"io"
)

// A Scene holds all of the contents of an STL file, as well as
// derived values.
type Scene struct {
	Header    [80]byte
	Triangles []Triangle
	Bounds    Subspace
}

// A Region is a region of the coordinate space from Min to Max
type Subspace struct {
	Min    Point
	Max    Point
}

// A Line is a line segment in 3D space, denoted by two Points
type Line struct {
	End [2]Point
}

// A Triangle is a triangle in 3D space, denoted by three Points.
// The "outside" of the triangle is denoted by its normal vector.
type Triangle struct {
	Normal Vector
	Vertex [3]Point
	Attr   uint16
}

// A Point is a point in 3D space.
type Point struct {
	X, Y, Z float32
}

func (p1 Point) Equals(p2 Point) bool {
	return p1.X == p2.X && p1.Y == p2.Y && p1.Z == p2.Z
}

func (s *Subspace) setmin(p Point) {
	if p.X < s.Min.X {
		s.Min.X = p.X
	}
	if p.Y < s.Min.Y {
		s.Min.Y = p.Y
	}
	if p.Z < s.Min.Z {
		s.Min.Z = p.Z
	}
}

func (s *Subspace) setmax(p Point) {
	if p.X > s.Max.X {
		s.Max.X = p.X
	}
	if p.Y > s.Max.Y {
		s.Max.Y = p.Y
	}
	if p.Z > s.Max.Z {
		s.Max.Z = p.Z
	}
}

type Vector struct {
	X, Y, Z float32
}

var epsilon float32 = 1e-5

func (v Vector) IsNormal() bool {
	return (1 - v.Abs()) < epsilon
}

func (v Vector) Abs() float32 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// short name, for convenience
var le = binary.LittleEndian

// Decode reads an entire STL file from a Reader, returning a Scene.
func Decode(r io.Reader) (s Scene, err error) {
	_, err = io.ReadFull(r, s.Header[:])
	if err != nil {
		return
	}

	var num uint32
	err = binary.Read(r, le, &num)
	if err != nil {
		return
	}

	s.Triangles = make([]Triangle, num)
	for i := uint32(0); i < num; i++ {
		err = readTriangle(r, &s.Triangles[i])
		if err != nil {
			return
		}

		// update the bounds
		for _, p := range s.Triangles[i].Vertex {
			s.Bounds.setmin(p)
			s.Bounds.setmax(p)
		}
	}

	return
}

// minimize copies by writing directly into the final storage
// place for the Triangle
func readTriangle(r io.Reader, t *Triangle) error {
	// read it into a buffer first, so that we can check err once
	var buf [50]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return err
	}

	// replace reader with one reading from buf
	r = bytes.NewReader(buf[:])

	binary.Read(r, le, &t.Normal.X)
	binary.Read(r, le, &t.Normal.Y)
	binary.Read(r, le, &t.Normal.Z)
	binary.Read(r, le, &t.Vertex[0].X)
	binary.Read(r, le, &t.Vertex[0].Y)
	binary.Read(r, le, &t.Vertex[0].Z)
	binary.Read(r, le, &t.Vertex[1].X)
	binary.Read(r, le, &t.Vertex[1].Y)
	binary.Read(r, le, &t.Vertex[1].Z)
	binary.Read(r, le, &t.Vertex[2].X)
	binary.Read(r, le, &t.Vertex[2].Y)
	binary.Read(r, le, &t.Vertex[2].Z)
	binary.Read(r, le, &t.Attr)

	return nil
}

func formatTriangle(t *Triangle) []byte {
	buf := &bytes.Buffer{}

	binary.Write(buf, le, &t.Normal.X)
	binary.Write(buf, le, &t.Normal.Y)
	binary.Write(buf, le, &t.Normal.Z)
	binary.Write(buf, le, &t.Vertex[0].X)
	binary.Write(buf, le, &t.Vertex[0].Y)
	binary.Write(buf, le, &t.Vertex[0].Z)
	binary.Write(buf, le, &t.Vertex[1].X)
	binary.Write(buf, le, &t.Vertex[1].Y)
	binary.Write(buf, le, &t.Vertex[1].Z)
	binary.Write(buf, le, &t.Vertex[2].X)
	binary.Write(buf, le, &t.Vertex[2].Y)
	binary.Write(buf, le, &t.Vertex[2].Z)
	binary.Write(buf, le, &t.Attr)

	return buf.Bytes()
}
