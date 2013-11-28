package main

import (
	"os"

	"code.google.com/p/jra-go/synth"
	"code.google.com/p/jra-go/wav"
)

func main() {
	w := new(wav.Wav)
	w.Channels = 1
	w.SamplesPerSecond = synth.SamplesPerSecond
	w.Samples[wav.Mono] = make([]wav.Sample8, w.SamplesPerSecond*5)

	freq := synth.Hertz(400)

	fs := synth.NewFrameSource()
	sw := synth.NewSquareWave(fs, synth.Hertz(400))
	r := synth.NewRenderer(sw, w)

	r.Render(
		// Render takes a callback that fires on every frame
		func() {
			// every frame, reduce the frequency down to 100
			// freq is a closure variable that survives
			// across calls
			if freq > 100 {
				freq -= 10
				sw.SetFrequency(freq)
			}
		})

	w.Write(os.Stdout)
}

// ex: ts=2
