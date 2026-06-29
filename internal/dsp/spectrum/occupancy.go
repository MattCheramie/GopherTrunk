package spectrum

import "math"

// Occupancy holds the spectral-occupancy measurements derived from a single
// FFT-shifted dBFS spectrum (a Frame's Bins). These are the lab-grade channel
// figures a vector-signal analyzer reports alongside the constellation: how
// wide the signal really is, how much power sits in the channel, how much leaks
// into the neighbours, and how tone-like vs noise-like the channel is.
type Occupancy struct {
	// OccupiedBandwidthHz is the ITU 99% occupied bandwidth: the span between
	// the points where the cumulative power crosses 0.5% and 99.5% of the
	// total, i.e. the band holding 99% of the power with 0.5% in each tail.
	OccupiedBandwidthHz float64 `json:"occupied_bandwidth_hz"`
	// ChannelPowerDBFS is the integrated power within ±channelBW/2 of DC,
	// in dBFS (10·log10 of the summed linear bin power).
	ChannelPowerDBFS float64 `json:"channel_power_dbfs"`
	// ACPRUpperDB / ACPRLowerDB are the adjacent-channel power ratios for the
	// upper/lower adjacent channels relative to the main channel, in dB
	// (negative for a clean signal that does not splatter into its neighbours).
	ACPRUpperDB float64 `json:"acpr_upper_db"`
	ACPRLowerDB float64 `json:"acpr_lower_db"`
	// SpectralFlatness is the Wiener entropy (geometric-mean / arithmetic-mean
	// of the in-channel linear power), 0..1: ~1 for white noise, ≪1 for a tone
	// or a narrow carrier.
	SpectralFlatness float64 `json:"spectral_flatness"`
}

// powerFloor clamps linear power away from zero so log/geomean math stays
// finite, matching the package's small-value clamp convention.
const powerFloor = 1e-30

// ComputeOccupancy derives the spectral-occupancy metrics from FFT-shifted dBFS
// bins (bin 0 = -sampleRate/2, DC at len/2, the order spectrum.AverageDB
// produces with centerHz=0). channelBWHz is the main channel width; adjOffsetHz
// is the centre spacing to each adjacent channel and adjBWHz its width — pass
// the protocol channel spacing for both to measure ACPR into the neighbouring
// channels. Non-positive widths fall back to sensible spans so a caller can ask
// for occupied bandwidth alone.
func ComputeOccupancy(binsDB []float32, sampleRateHz, channelBWHz, adjOffsetHz, adjBWHz float64) Occupancy {
	n := len(binsDB)
	if n == 0 || sampleRateHz <= 0 {
		return Occupancy{}
	}
	binHz := sampleRateHz / float64(n)
	baseHz := -sampleRateHz / 2 // frequency of bin 0

	// Linear power per bin (floored).
	lin := make([]float64, n)
	var total float64
	for i, db := range binsDB {
		p := dbToLinearPower(float64(db))
		if p < powerFloor {
			p = powerFloor
		}
		lin[i] = p
		total += p
	}

	freqOf := func(i int) float64 { return baseHz + float64(i)*binHz }

	// bandPower sums the linear power of bins whose centre frequency falls in
	// [loHz, hiHz], also returning the bin count for flatness.
	bandPower := func(loHz, hiHz float64) (sum float64, count int) {
		for i := 0; i < n; i++ {
			f := freqOf(i)
			if f >= loHz && f <= hiHz {
				sum += lin[i]
				count++
			}
		}
		return sum, count
	}

	out := Occupancy{}

	// 99% occupied bandwidth: cumulative power, 0.5% tail on each side.
	if total > 0 {
		loTarget := 0.005 * total
		hiTarget := 0.995 * total
		var cum float64
		loBin, hiBin := 0, n-1
		loSet := false
		for i := 0; i < n; i++ {
			cum += lin[i]
			if !loSet && cum >= loTarget {
				loBin = i
				loSet = true
			}
			if cum >= hiTarget {
				hiBin = i
				break
			}
		}
		out.OccupiedBandwidthHz = float64(hiBin-loBin) * binHz
	}

	// Main channel power.
	chBW := channelBWHz
	if chBW <= 0 {
		chBW = sampleRateHz // whole span
	}
	mainPow, mainCount := bandPower(-chBW/2, chBW/2)
	out.ChannelPowerDBFS = 10 * math.Log10(math.Max(mainPow, powerFloor))

	// ACPR into the upper/lower adjacent channels.
	if adjOffsetHz > 0 && adjBWHz > 0 && mainPow > 0 {
		upPow, _ := bandPower(adjOffsetHz-adjBWHz/2, adjOffsetHz+adjBWHz/2)
		loPow, _ := bandPower(-adjOffsetHz-adjBWHz/2, -adjOffsetHz+adjBWHz/2)
		out.ACPRUpperDB = 10 * math.Log10(math.Max(upPow, powerFloor)/mainPow)
		out.ACPRLowerDB = 10 * math.Log10(math.Max(loPow, powerFloor)/mainPow)
	}

	// Spectral flatness (Wiener entropy) over the in-channel bins.
	if mainCount > 0 {
		var lnSum, arithSum float64
		for i := 0; i < n; i++ {
			f := freqOf(i)
			if f < -chBW/2 || f > chBW/2 {
				continue
			}
			lnSum += math.Log(lin[i])
			arithSum += lin[i]
		}
		geoMean := math.Exp(lnSum / float64(mainCount))
		arithMean := arithSum / float64(mainCount)
		if arithMean > 0 {
			out.SpectralFlatness = geoMean / arithMean
		}
	}

	return out
}
