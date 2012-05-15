package bcd

/*
typedef unsigned short int ushort;

ushort nativeAdd(ushort a, ushort b) {
	ushort c;
        asm ("add %%dl, %%al; daa"
             :"=a"(c)
             :"d"(b),"a"(a)
             );
	return c;
}

*/
import "C"

import (
	"fmt"
)

type Bcd int

func ToBcd(a uint) Bcd {
	if (a > 9999) {
		panic("number too high for 16-bit BCD")
	}
	r := uint(0)
	shift := uint(0)
	for a > 0 {
		r += (a % 10) << shift
		shift += 4
		a = a / 10
	}
	return Bcd(r)
}

func (b Bcd) UInt() uint {
	r := uint(0)
	mult := uint(1)
	mask := uint(0xf)
	shift := uint(0)

	for i := 0; i < 4; i++ {
		digit := (uint(b) & mask) >> shift
		if digit > 9 {
			panic("bad digit in bcd number")
		}
		r += digit * mult

		mult *= 10
		shift += 4
		mask = mask << 4
	}

	return r
}

func (b Bcd) String() string {
	return fmt.Sprintf("%v", b.UInt())
}

func goAdd(a, b Bcd) Bcd {
	shift := uint(0)
	mask := uint(0xf)
	r := uint(0)
	carry := uint(0)
	for i := 0; i < 4; i++ {
		x := carry + ((uint(a) & mask) >> shift) + ((uint(b) & mask) >> shift)
		if x > 9 {
			x += 6
		}
		r += (x & 0xf) << shift
		if x > 15 {
			carry = 1
		} else {
			carry = 0
		}
		shift += 4
		mask = mask << 4
	}
	if r > 0x9999 {
		panic("overflow in add")
	}
	return Bcd(r)
}

func AddNative(a, b Bcd) Bcd {
	return Bcd(C.nativeAdd(C.ushort(a), C.ushort(b)))
}

func AddGo(a, b Bcd) Bcd {
	return goAdd(a,b)
}
