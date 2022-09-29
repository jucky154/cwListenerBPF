/*
MIT License
Copyright (c) 2022 JA1ZLO
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"github.com/mjibson/go-dsp/spectral"
	"github.com/mjibson/go-dsp/wav"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"github.com/thoas/go-funk"
	"math"
	"log"
	"os"
	"sort"
	"fmt"
)

type XYs []XY

type XY struct {
	X, Y float64
}



func LPF(source []float64, n int) (result []float64) {
	result = make([]float64, len(source)-n)
	ave := float64(0.0)

	for i := 0; i < n; i++ {
		ave += source[i] / float64(n)
	}

	result[0] = ave

	for i := 1; i < len(result); i++ {
		neg := source[i+0] / float64(n)
		pos := source[i+n] / float64(n)
		ave = ave - neg + pos
		result[i] = ave
	}

	return
}

func PeakFreq(signal []float64, sampling_freq uint32)([]float64, []float64) {
	var opt spectral.PwelchOptions

	opt.NFFT = 4096
	opt.Noverlap = 1024
	opt.Window = nil
	opt.Pad = 4096
	opt.Scale_off = false

	Power, Freq := spectral.Pwelch(signal, float64(sampling_freq), &opt)

	peakPower := 0.0
	powerarr := make([]float64, 0)
	freqarr := make([]float64, 0)
	for i, val := range Freq {
		if val > 200 && val < 2000 {
		       powerarr = append(powerarr, Power[i])
		       freqarr = append(freqarr, val)
			if Power[i] > peakPower {
				peakPower = Power[i]
			}
		}
	}

	return powerarr, freqarr
}

func BPF(input []float64, samplerate float64, freq float64, bw float64) []float64{
     omega := float64(2.0) * math.Pi * freq / samplerate
     alpha := math.Sin(omega) * math.Sinh(math.Log(float64(2.0))*float64(0.5)*bw*omega/math.Sin(omega))

     a0 := float64(1.0) + alpha
     a1 := -float64(2.0) * math.Cos(omega)
     a2 := float64(1.0) - alpha
     b0 := alpha
     b1 := float64(0.0)
     b2 := -alpha

     in1 := float64(0.0)
     in2 := float64(0.0)
     out1 := float64(0.0)
     out2 := float64(0.0)

     output := make([]float64, len(input))

     for i, val := range(input) {
     	 output[i] = b0/a0 * val + b1/a0 * in1 + b2/a0 * in2 -a1/a0 * out1 - a2/a0 * out2
	 
	 in2 = in1
	 in1 = val

	 out2 = out1
	 out1 = output[i]
     }

     return output
}

type PeakFFT struct {
     index int
     power float64
}
     

func DetectPeakFFT(threshold float64, y []float64) (result []PeakFFT) {
     	peak_value := funk.MaxFloat64(y)
	delta := peak_value * threshold

	mn := peak_value * float64(2.0)
	mx := float64(-1)
	var mxpos int
	result = make([]PeakFFT, 0)
	var buf PeakFFT

	lookformax := true

	for i, this := range y {
		if this > mx {
			mx = this
			mxpos = i
		}

		if this < mn {
			mn = this
		}

		if lookformax {
			if this < mx-delta {
				buf.index = mxpos
				buf.power = mx
				result = append(result, buf)
				mn = this
				lookformax = false
			}
		} else {
			if this > mn+delta {
				mx = this
				mxpos = i
				lookformax = true
			}
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].power > result[j].power })

	return
}

	

func main() {
	file, err := os.Open("JA1ZLO.wav")
	if err != nil {
		log.Fatal(err)
	}

	w, werr := wav.New(file)
	if werr != nil {
		log.Fatal(werr)
	}

	len_sound := w.Samples
	rate_sound := w.SampleRate
	SoundData, werr := w.ReadFloats(len_sound)
	if werr != nil {
		log.Fatal(werr)
	}

	Signal64 := make([]float64, len_sound)
	SquaredSignal64 := make([]float64, len_sound)
	for i, val := range SoundData {
		Signal64[i] = float64(val)
		SquaredSignal64[i] = float64(val) * float64(val)
	}

	powerarr, freqarr := PeakFreq(Signal64, rate_sound)
	test := DetectPeakFFT(float64(0.5), powerarr)
	
	ave_num := 6 * int(float64(rate_sound)/freqarr[test[0].index])

	bpf_freq := freqarr[test[0].index]
	smoothed := BPF(SquaredSignal64, float64(rate_sound), bpf_freq, float64(0.1))
	smoothed = LPF(LPF(LPF(LPF(smoothed, ave_num), ave_num), ave_num), ave_num)
	fmt.Println(freqarr[test[0].index])
	
	pts := make(plotter.XYs, len(smoothed))

	for i, val := range smoothed {
		pts[i].X = float64(i) / float64(rate_sound)
		pts[i].Y = val
	}


	p := plot.New()

	p.Title.Text = "signal power"
	p.X.Label.Text = "t"
	p.Y.Label.Text = "power"

	plotutil.AddLines(p, pts)
	p.Save(10*vg.Inch, 3*vg.Inch, "smoothed.png")
}