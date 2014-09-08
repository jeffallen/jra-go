package stl

import (
	"bytes"
	"strings"
	"testing"
)

/*
func TestTriangleSlice(t *testing.T) {
	tri := Triangle{
		Vector{1, 1, 1},
		[3]Point{Point{0, 0, 0},
			Point{0, 1, 0},
			Point{1, 0, 1},
		}, 0,
	}
	intsersect, line := tri.Slice(Point{0,0,.5})
	if !intersect {
		t.Error("expected intersection")
	}
	t.Log(line)
}
*/

func TestVector(t *testing.T) {
	v := Vector{0, 0, 1}
	if !v.IsNormal() {
		t.Error("should be normal")
	}
	if v.Abs() != 1.0 {
		t.Error("abs is wrong")
	}
}

func TestStlReader(t *testing.T) {
	data := []byte{
		// header
		'C', 'O', 'L', 'O', 'R', '=', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 16
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 32
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 64
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 80
		1, 0, 0, 0, // 80: 4 bytes, little endian, how many triangles
		// 84: triangle
	}

	tri := Triangle{
		Vector{1, 1, 1},
		[3]Point{Point{0, 0, 0},
			Point{1, 2, 3},
			Point{3, 2, -1},
		}, 0,
	}
	data = append(data, formatTriangle(&tri)...)

	s, err := Decode(bytes.NewReader(data[0:10]))
	if err == nil {
		t.Error("should have gotten an error")
	}

	s, err = Decode(bytes.NewReader(data))
	if err != nil {
		t.Error("got err", err)
	}
	if !strings.Contains(string(s.Header[:]), "COLOR=") {
		t.Error("header is missing color=")
	}
	if len(s.Triangles) != 1 {
		t.Errorf("expected 1 triangle, got %d", len(s.Triangles))
	}

	if !s.Bounds.Min.Equals(Point{0, 0, -1}) {
		t.Error("bad bounds min", s.Bounds.Min)
	}
	if !s.Bounds.Max.Equals(Point{3, 2, 3}) {
		t.Error("bad bounds max", s.Bounds.Max)
	}
}
