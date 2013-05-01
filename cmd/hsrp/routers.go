package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// taken from the HSRP spec (as decoded by Wireshark, anyway)
type hsrpHello struct {
	Version        byte
	OpCode         byte
	State          byte
	Hellotime      byte
	Holdtime       byte
	Priority       byte
	Group          byte
	Reserved       byte
	Authentication [8]byte
	Vip            [4]byte
}

type neighbors struct {
	mu sync.Mutex
	m  map[string]net.IP
}

func (n *neighbors) add(a net.Addr, v net.IP) {
	n.mu.Lock()
	n.m[a.String()] = v
	n.mu.Unlock()
}

func (n *neighbors) dump() {
	fmt.Println()
	fmt.Println("Known HSRP routers:")

	n.mu.Lock()
	for k, v := range n.m {
		fmt.Println("\t", k, "vip:", v)
	}
	n.mu.Unlock()
}

func newNeighbors() *neighbors {
	return &neighbors{m: make(map[string]net.IP)}
}

func main() {
	n := newNeighbors()

	// a goroutine to listen for HSRP hellos
	go func() {
		// 1985 = HSRP port
		group, err := net.ResolveUDPAddr("udp", "224.0.0.2:1985")
		if err != nil {
			fmt.Println("error from ResolveUDPAddr:", err)
			return
		}

		c, err := net.ListenMulticastUDP("udp", nil, group)
		if err != nil {
			fmt.Println("error from ListenMulticastUDP:", err)
			return
		}

		for {
			var buf [4096]byte
			l, addr, err := c.ReadFrom(buf[:])
			if err != nil {
				fmt.Println("error from ReadFrom:", err)
			}

			// is this a hello packet?
			if l > 2 && buf[0] == 0 && buf[1] == 0 {
				hello := &hsrpHello{}
				err := binary.Read(bytes.NewBuffer(buf[0:l]), binary.BigEndian, hello)
				if err != nil {
					fmt.Println("error from binary.Read:", err)
					continue
				}
				vip := net.IP(hello.Vip[:])
				n.add(addr, vip)
			}
		}
	}()

	for {
		n.dump()
		time.Sleep(1 * time.Second)
	}
}
