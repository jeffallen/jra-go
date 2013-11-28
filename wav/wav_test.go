package wav

import (
	"bytes"
	"testing"
)

func TestWav(t *testing.T) {
	w := new(Wav)
	w.Channels = 1
	w.SamplesPerSecond = 44100
	w.Samples[Mono] = make([]Sample8, w.SamplesPerSecond*5)

	buf := new(bytes.Buffer)
	w.Write(buf)
	t.Log("Output:", buf)
}

// ex: ts=2
