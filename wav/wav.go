package wav

import (
	"encoding/binary"
	"io"
)

type Wav struct {
	SamplesPerSecond int
	Channels         int
	BitsPerSample    int
	Samples          [2][]int16 // maximum 2 channels
}

const Mono = 0
const Left = 0
const Right = 1

func (w *Wav) Write(wr io.Writer) (err error) {
	// TODO: need to calculate this
	len := int32(16)

	_, err = wr.Write([]byte("RIFF"))
	if err != nil { return }

	err = binary.Write(wr, binary.LittleEndian, len)
	if err != nil { return }

	_, err = wr.Write([]byte("WAVE"))
	if err != nil { return }

	err = w.writeFmt(wr)
	if err != nil { return }

	err = w.writeSamples(wr)
	if err != nil { return }

	return
}

func (w *Wav) writeSamples(wr io.Writer) (err error) {
	if _, err = wr.Write([]byte("data")); err != nil { return }

	if w.Channels > 1 {
		if len(w.Samples[Left]) != len(w.Samples[Right]) {
			panic("samples for left and right not the same length")
		}
	}

	l := w.Channels * len(w.Samples[Left]) * w.BitsPerSample / 8
	// The sample data must end on an even byte boundary
	oneMore := false
	if l & 1 != 0 {
		l++
		oneMore = true
	}

	// 4 bytes  <length of the data block>
	if err = binary.Write(wr, binary.LittleEndian, int32(l)); err != nil { return }

	for i := 0; i < len(w.Samples[Left]); i++ {
		for ch := 0; ch < w.Channels; ch++ {
			if w.BitsPerSample == 8 {
				err = binary.Write(wr, binary.LittleEndian, uint8(w.Samples[ch][i]))
			} else {
				err = binary.Write(wr, binary.LittleEndian, int16(w.Samples[ch][i]))
			}
		}
	}

	// The sample data must end on an even byte boundary
	if oneMore {
		err = binary.Write(wr, binary.LittleEndian, uint8(128))
	}

	return
}

func (w *Wav) writeFmt(wr io.Writer) (err error) {
	if w.BitsPerSample != 16 && w.BitsPerSample != 8 {
		panic("bad bits per sample")
	}

	blockAlign := w.Channels * w.BitsPerSample / 8
	bytesPerSecond := blockAlign * w.SamplesPerSecond

	if _, err = wr.Write([]byte("fmt ")); err != nil { return }

	// 4 bytes  0x00000010     // Length of the fmt data (16 bytes)
	if err = binary.Write(wr, binary.LittleEndian, int32(16)); err != nil {
		return
	}

	// 2 bytes  0x0001         // Format tag: 1 = PCM
	if err = binary.Write(wr, binary.LittleEndian, int16(1)); err != nil {
		return
	}

	// 2 bytes  <channels>     // Channels: 1 = mono, 2 = stereo
	if err = binary.Write(wr, binary.LittleEndian, int16(1)); err != nil {
		return
	}

	// 4 bytes  <sample rate>  // Samples per second: e.g., 44100
	if err = binary.Write(wr, binary.LittleEndian, int32(w.SamplesPerSecond)); err != nil {
		return
	}

	// 4 bytes  <bytes/second> // sample rate * block align
	if err = binary.Write(wr, binary.LittleEndian, int32(bytesPerSecond)); err != nil {
		return
	}

	// 2 bytes  <block align>  // channels * bits/sample / 8
	if err = binary.Write(wr, binary.LittleEndian, int16(blockAlign)); err != nil {
		return
	}

	// 2 bytes  <bits/sample>  // 8 or 16
	if err = binary.Write(wr, binary.LittleEndian, int16(w.BitsPerSample)); err != nil {
		return
	}

	return
}

// ex: ts=2
