package synth

import (
	"testing"

	"code.google.com/p/jra-go/wav"
)

func TestRender8(t *testing.T) {
	expected := []wav.Sample8{ 0, 64, 128, 192, 255 }
	f := new(Frame)
	f[0] = -1
	f[1] = -0.5
	f[2] = 0
	f[3] = 0.5
	f[4] = 1

	g := make([]wav.Sample8, len(f))
	f.RenderTo8Bit(g)
	t.Log("Output:", g[0:5])
	for i,expect := range expected {
		if g[i] != expect {
			t.Fatalf("index %v: %v != %v", i, g[i], expect)
		}
	}
}

// ex: ts=2
