package synth

import (
	"code.google.com/p/jra-go/wav"
)

// current limits: always runs at 44.1 khz, always processes
// frames of 100 ms

const SamplesPerSecond = 44100
const frameTime = 100 / Sample(1000) // 100 ms
const frameLength = int(SamplesPerSecond * frameTime)

type Sample float32
type Hertz float32
type Frame [frameLength]Sample

type Pluggable interface {
	GetOutput() chan *Frame
}

type BasicBlock struct {
	in, out chan *Frame
}

func (b BasicBlock) GetOutput() chan *Frame {
	if b.out == nil {
		panic("Attempt to GetOutput on an uninitialized block.")
	}
	return b.out
}

func (b *BasicBlock) init(src Pluggable) {
	// src == nil means that this block is a source (i.e. FrameSource)
	if src != nil {
		b.in = src.GetOutput()
	}
	b.out = make(chan *Frame)
}

func (b *BasicBlock) done(f *Frame) {
	b.out <- f
}

type Const struct {
	BasicBlock
	Level Sample
}

func NewConst(src Pluggable, level Sample) (c *Const) {
	c = new(Const); c.init(src)
	c.Level = level

	go func() {
		for f := range c.in {
			for i := 0; i < len(f); i++ {
				f[i] = c.Level
			}
			c.done(f)
		}
	}()
	return
}

type SquareWave struct {
	BasicBlock
	halfwave uint
}

func NewSquareWave(src Pluggable, freq Hertz) (s *SquareWave) {
	s = new(SquareWave); s.init(src)
	s.SetFrequency(freq)
	
	go func() {
		val := Sample(1.0)
		// it's ok if cur wraps, as long as its period is > halfwave
		cur := uint(0)
		for f := range s.in {
			for i := 0; i < len(f); i++ {
				if cur % s.halfwave == 0 {
					val = 1
				} else {
					val = -1
				}
				f[i] = val
				cur++
			}
			s.done(f)
		}
	}()

	return
}

func (sw *SquareWave)SetFrequency(freq Hertz) {
	sw.halfwave = uint(SamplesPerSecond/freq/2)
}

type Renderer struct {
	BasicBlock
	wav *wav.Wav
}

func NewRenderer(src Pluggable, wav *wav.Wav) (r *Renderer) {
	r = new(Renderer)
	// no call to r.init() because a renderer has not output
	r.in = src.GetOutput()
	r.wav = wav
	return
}

func (r *Renderer)Render(cb func()) {
	w := r.wav
	for i := 0; i < len(w.Samples[wav.Mono]); {
		frame := <-r.in
		if cb != nil { cb() }
		frame.RenderTo8Bit(w.Samples[wav.Mono][i:i+len(frame)])
		i += len(frame)
	}
}

func (r *Renderer)GetOutput() chan *Frame {
	panic("Do not call GetOutput on a Renderer.")
	return nil
}

type FrameSource struct {
	BasicBlock
}

func NewFrameSource() (fs *FrameSource) {
	fs = new(FrameSource); fs.init(nil)

	go func() {
		for {
			f := new(Frame)
			fs.done(f)
		}
	}()
	return
}

func (f *Frame) RenderTo8Bit(out []wav.Sample8) {
	if len(out) != len(f) {
		panic("mismatched buffers")
	}
	for i := 0; i < len(f); i++ {
		in := f[i]

		// clip at [-1.0, 1.0]
		if in > 1.0 {
			in = 1.0
		}
		if in < -1.0 {
			in = -1.0
		}

		if in == 1.0 {
			// special case because 1.0 -> 0 otherwise (due to overflow)
			out[i] = 255
		} else {
			out[i] = wav.Sample8((in + 1) / 2 * 256)
		}
	}
}

// ex:ts=2
