package bcd

import (
	"fmt"
	"testing"
)

func TestToBcd(t *testing.T) {
	b := ToBcd(101)
	if uint(b) != 0x101 {
		t.Errorf("Got bcd %x, expected %x", uint(b), 0x101)
	}

	s := fmt.Sprintf("%v", b)
	if s != "101" {
		t.Errorf("Got string representation %v, expected %v", s, "101")
	}
}

func TestAdds(t *testing.T) {
	a := []Bcd{  ToBcd(1), ToBcd(9), ToBcd(10), ToBcd(9998) }
	b := []Bcd{  ToBcd(1), ToBcd(1), ToBcd(1),  ToBcd(1) }
	c := []uint{ 2,        10,       11,        9999 }

	for i,_ := range a {
		d := AddNative(a[i],b[i]).UInt()
		if c[i] != d {
			t.Errorf("AddNative: %d + %d != %d (got %d)", a[i].UInt(), b[i].UInt(), c[i], d)
		}
		d = AddGo(a[i],b[i]).UInt()
		if c[i] != d {
			t.Errorf("AddGo: %d + %d != %d (got %d)", a[i].UInt(), b[i].UInt(), c[i], d)
		}
	}
}

var bcds []Bcd

func init() {
	bcds = make([]Bcd, 20)
	for i := 0; i < len(bcds); i++ {
		bcds[i] = Bcd(uint(i))
	}
}

func BenchmarkAddGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		j := i % len(bcds)
		k := (i+1) % len(bcds)
		_ = AddGo(bcds[j], bcds[k])
	}
}

func BenchmarkAddNative(b *testing.B) {
	for i := 0; i < b.N; i++ {
		j := i % len(bcds)
		k := (i+1) % len(bcds)
		_ = AddNative(bcds[j], bcds[k])
	}
}
