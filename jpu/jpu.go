// This package implements Jeff's Processing Unit, an
// 8-bit, memory-mapped IO processor.
package jpu

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const NumReg int = 10

type Instruction byte

const (
	InsHalt Instruction = iota // halt = 0, so that uninit memory (0) causes a halt
	InsNop
	InsMemReg
	InsRegMem
	InsImmReg
	InsMovReg
	InsAddReg
	InsSubReg
	InsDivReg
	InsGotoIfEqual
	InsGotoIfNotEqual
	//InsCall
	//InsReturn
	InsWait
)

type Address uint16
type Registers [NumReg]Address
type ReadCallback func(where Address) byte
type WriteCallback func(where Address, what byte)

type Processor struct {
	mem       []byte
	Top       int
	Reg       Registers
	input     map[Address]ReadCallback
	output    map[Address]WriteCallback
	traceName string
	logger    Logger
}

func NewProcessor(ram int) *Processor {
	return &Processor{
		mem:    make([]byte, ram),
		Top:    ram,
		input:  make(map[Address]ReadCallback),
		output: make(map[Address]WriteCallback),
	}
}

// Registers a callback for a read on a memory mapped IO
// location.
func (p *Processor) RegisterIn(fp ReadCallback, where Address) {
	if _, exists := p.input[where]; exists {
		panic("cannot put two callbacks on the same memory location")
	} else {
		p.input[where] = fp
	}
}

// Registers a callback for a write on a memory mapped IO
// location.
func (p *Processor) RegisterOut(fp WriteCallback, where Address) {
	if _, exists := p.output[where]; exists {
		panic("cannot put two callbacks on the same memory location")
	} else {
		p.output[where] = fp
	}
}

func (p *Processor) addressInRange(where Address) bool {
	return where < Address(len(p.mem))
}

func (p *Processor) Peek(where Address) byte {
	if !p.addressInRange(where) {
		p.trace(fmt.Sprintf("read from address %d out of range", where))
		return 0
	}
	if fp, exists := p.input[where]; exists {
		return fp(where)
	}
	return p.mem[where]
}

func (p *Processor) Poke(where Address, what byte) {
	if !p.addressInRange(where) {
		p.trace(fmt.Sprintf("write to address %d out of range", where))
	} else {
		if fp, exists := p.output[where]; exists {
			fp(where, what)
		} else {
			p.mem[where] = what
		}
	}
}

func (p *Processor) getAddress(where Address) (res Address) {
	high := Address(p.Peek(where))
	low := Address(p.Peek(where + 1))
	res = (high << 8) + low
	return
}

func (p *Processor) LoadMem(program []byte, where Address) {
	for i := 0; i < len(program); i++ {
		p.Poke(where+Address(i), program[i])
	}
}

func (p *Processor) trace(msg string) {
	if p.traceName != "" {
		p.logger.Log(fmt.Sprintf("%s: %s", p.traceName, msg))
	}
}

type Logger interface {
	Log(msg string)
}

func (p *Processor) Trace(name string, l Logger) {
	p.traceName = name
	p.logger = l
}

func (p *Processor) getReg(reg byte) Address {
	if int(reg) > len(p.Reg) {
		return 0
	}
	return p.Reg[reg]
}

func (p *Processor) StepN(steps int) bool {
	for steps > 0 {
		if !p.Step() {
			return false
		}
		steps--
	}
	return true
}

// Run the processor for one step. If the processor does not halt in this
// step, returns true.
func (p *Processor) Step() bool {
	ip := p.Reg[0]
	// ip(original) for logging
	ipo := ip
	instruction := Instruction(p.Peek(ip))
	ip++

	switch instruction {
	default:
		p.trace(fmt.Sprintf("%d: unknown (treated as nop)", ipo))
	case InsNop:
		p.trace(fmt.Sprintf("%d: nop", ipo))
	case InsMemReg:
		from := p.Peek(ip)
		ip++
		where := p.getReg(from)
		to := p.Peek(ip)
		ip++
		p.trace(fmt.Sprintf("%d: *r%d -> r%d", ipo, from, to))
		p.Reg[to] = Address(p.Peek(where))
		if to == 0 {
			// do not do final Reg[0]=ip if this instruction is a goto
			return true
		}
	case InsRegMem:
		from := p.Peek(ip)
		ip++
		to := p.Peek(ip)
		ip++
		where := p.getReg(to)
		p.trace(fmt.Sprintf("%d: r%d -> *r%d", ipo, from, to))
		p.Poke(where, byte(p.Reg[from]&0xff))
	case InsImmReg:
		imm := p.getAddress(ip)
		ip = ip + 2
		to := p.Peek(ip)
		ip++
		p.trace(fmt.Sprintf("%d: value %d -> r%d", ipo, imm, to))
		p.Reg[to] = imm
		if to == 0 {
			// do not do final Reg[0]=ip if this instruction is a goto
			return true
		}
	case InsGotoIfEqual:
		where := p.getAddress(ip)
		ip = ip + 2
		ar := p.Peek(ip)
		ip++
		br := p.Peek(ip)
		ip++
		p.trace(fmt.Sprintf("%d: goto %d if r%d == r%d", ipo, where, ar, br))
		if p.getReg(ar) == p.getReg(br) {
			p.Reg[0] = where
			// do not do final reg[0]=ip
			return true
		}
	case InsGotoIfNotEqual:
		where := p.getAddress(ip)
		ip = ip + 2
		ar := p.Peek(ip)
		ip++
		br := p.Peek(ip)
		ip++
		p.trace(fmt.Sprintf("%d: goto %d if r%d != r%d", ipo, where, ar, br))
		if p.getReg(ar) != p.getReg(br) {
			p.Reg[0] = where
			// do not do final reg[0]=ip
			return true
		}
	case InsAddReg:
		b := p.Peek(ip)
		ip++
		a := p.Peek(ip)
		ip++
		if int(a) < len(p.Reg) && int(b) < len(p.Reg) {
			p.Reg[a] = p.Reg[a] + p.Reg[b]
		} else {
			panic("bad register")
		}
		p.trace(fmt.Sprintf("%d: r%d + r%d -> r%d", ipo, a, b, a))
		if a == 0 {
			// do not do final reg[0]=ip
			return true
		}
	case InsSubReg:
		b := p.Peek(ip)
		ip++
		a := p.Peek(ip)
		ip++
		if int(a) < len(p.Reg) && int(b) < len(p.Reg) {
			p.Reg[a] = p.Reg[a] - p.Reg[b]
		} else {
			panic("bad register")
		}
		p.trace(fmt.Sprintf("%d: r%d - r%d -> r%d", ipo, a, b, a))
		if a == 0 {
			// do not do final reg[0]=ip
			return true
		}
	case InsDivReg:
		b := p.Peek(ip)
		ip++
		a := p.Peek(ip)
		ip++
		if int(a) < len(p.Reg) && int(b) < len(p.Reg) {
			p.Reg[a] = p.Reg[a] / p.Reg[b]
		} else {
			panic("bad register")
		}
		p.trace(fmt.Sprintf("%d: r%d / r%d -> r%d", ipo, a, b, a))
		if a == 0 {
			// do not do final reg[0]=ip
			return true
		}
	case InsMovReg:
		b := p.Peek(ip)
		ip++
		a := p.Peek(ip)
		ip++
		if int(a) < len(p.Reg) && int(b) < len(p.Reg) {
			p.Reg[a] = p.Reg[b]
		} else {
			panic("bad register")
		}
		p.trace(fmt.Sprintf("%d: r%d -> r%d", ipo, b, a))
		if a == 0 {
			// do not do final reg[0]=ip
			return true
		}
		/*

			Taking these out for now, because they are not needed
			and confusing for clue solvers to have to discover and not use.

					case InsCall:
						where := p.getAddress(ip)
						ip = ip + 2
						p.Reg[9] -= 2
						stack := p.Reg[9]
						p.Poke(stack, byte((ip>>8)&0xff))
						p.Poke(stack+1, byte(ip&0xff))
						p.Reg[0] = where
						p.trace(fmt.Sprintf("%d: call %d # new top of stack: %d", ipo, where, stack))
						return true
					case InsReturn:
						stack := p.Reg[9]
						to := p.getAddress(stack)
						p.Reg[9] += 2
						p.Reg[0] = to
						p.trace(fmt.Sprintf("%d: return # new top of stack: %d", ipo, p.Reg[9]))
						return true
		*/
	case InsHalt:
		p.trace(fmt.Sprintf("%d: halt", ipo))
		p.Reg[0] = ip
		return false
	}
	p.Reg[0] = ip
	return true
}

func getAddr(pass int, s string, labels map[string]Address) (val Address) {
	s = strings.ToLower(s)
	if len(s) > 1 && strings.ContainsRune("abcdefghijklmnopqrstuvwxyz", rune(s[0])) {
		var ok bool
		if val, ok = labels[s]; !ok {
			if pass == 0 {
				// during first pass, just return 0 for unk
				val = 0
				return
			} else {
				panic(fmt.Sprintf("unknown label %s", s))
			}
		}
	} else {
		i, err := strconv.ParseInt(s, 0, 16)
		if err != nil {
			panic(fmt.Sprintf("Failed to turn %s into a number (%s)", s, err))
		}
		val = Address(i)
	}
	return
}

func need(x []string, i int) {
	if len(x) < i+1 {
		panic(fmt.Sprintf("not enough arguments to %s", x[0]))
	}
}

func Assemble(prog string) (res []byte, org int) {
	labels := make(map[string]Address)
	pass := 0
reparse:
	org = 0
	here := Address(org)

	rd := bufio.NewReader(strings.NewReader(prog))
	for {
		s, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		// turn tabs into spaces
		r := strings.NewReplacer("\t", " ")
		s = r.Replace(s)
		// remove leading spaces
		s = strings.TrimLeft(s, " ")
		// remove trailing whitespace
		s = strings.TrimRight(s, " \n")

		// skip comments
		if strings.HasPrefix(s, "#") {
			continue
		}

		// look for label, remember this spot, remove it
		if colon := strings.Index(s, ":"); colon != -1 {
			label := s[0:colon]
			labels[label] = here
			s = s[colon+1:]
		}

		tok := strings.Split(s, " ")
		// empty line?
		if len(tok) < 1 || tok[0] == "" {
			continue
		}

		seenorg := false

		switch strings.ToLower(tok[0]) {
		default:
			panic(fmt.Sprintf("bad opcode %s", tok[0]))
		case "org":
			if pass == 0 && seenorg {
				panic("only one org allowed")
			}
			need(tok, 1)
			here = getAddr(pass, tok[1], labels)
			org = int(here)
			seenorg = true
		case "halt":
			res = append(res, byte(InsHalt))
			here++
		case "nop":
			res = append(res, byte(InsNop))
			here++
		case "wait":
			res = append(res, byte(InsWait))
			here++
			/*
				case "return":
					res = append(res, byte(InsReturn))
					here++
				case "call":
					need(tok, 1)
					res = append(res, byte(InsCall))
					val := getAddr(pass, tok[1], labels)
					res = append(res, byte((val >> 8) & 0xff))
					res = append(res, byte(val & 0xff))
					here += 3
			*/
		case "gotoifnotequal":
			need(tok, 3)
			res = append(res, byte(InsGotoIfNotEqual))
			val := getAddr(pass, tok[1], labels)
			res = append(res, byte((val>>8)&0xff))
			res = append(res, byte(val&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[3], labels)&0xff))
			here += 5
		case "gotoifequal":
			need(tok, 3)
			res = append(res, byte(InsGotoIfEqual))
			val := getAddr(pass, tok[1], labels)
			res = append(res, byte((val>>8)&0xff))
			res = append(res, byte(val&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[3], labels)&0xff))
			here += 5
		case "regmem":
			need(tok, 2)
			res = append(res, byte(InsRegMem))
			res = append(res, byte(getAddr(pass, tok[1], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 3
		case "memreg":
			need(tok, 2)
			res = append(res, byte(InsMemReg))
			res = append(res, byte(getAddr(pass, tok[1], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 3
		case "movreg":
			need(tok, 2)
			res = append(res, byte(InsMovReg))
			res = append(res, byte(getAddr(pass, tok[1], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 3
		case "addreg":
			need(tok, 2)
			res = append(res, byte(InsAddReg))
			res = append(res, byte(getAddr(pass, tok[1], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 3
		case "subreg":
			need(tok, 2)
			res = append(res, byte(InsSubReg))
			res = append(res, byte(getAddr(pass, tok[1], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 3
		case "divreg":
			need(tok, 2)
			res = append(res, byte(InsDivReg))
			res = append(res, byte(getAddr(pass, tok[1], labels)&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 3
		case "immreg":
			need(tok, 2)
			res = append(res, byte(InsImmReg))
			val := getAddr(pass, tok[1], labels)
			res = append(res, byte((val>>8)&0xff))
			res = append(res, byte(val&0xff))
			res = append(res, byte(getAddr(pass, tok[2], labels)&0xff))
			here += 4
		case "raw":
			for _, x := range tok {
				i, _ := strconv.ParseInt(x, 0, 8)
				res = append(res, byte(i))
				here++
			}
		}
	}
	// second pass to handle forward refs
	if pass == 0 {
		res = []byte{}
		pass++
		goto reparse
	}
	return
}
