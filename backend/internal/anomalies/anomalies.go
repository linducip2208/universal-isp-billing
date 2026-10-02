// Package anomalies: deterministic statistical detectors (z-score,
// threshold, spike/drop) over metric series. ML models plug in behind the
// Detector interface; no black boxes in the default path.
package anomalies

import "math"

type Point struct {
	At    int64   `json:"at"`
	Value float64 `json:"value"`
}

type Finding struct {
	Metric string  `json:"metric"`
	At     int64   `json:"at"`
	Value  float64 `json:"value"`
	ZScore float64 `json:"zscore"`
	Reason string  `json:"reason"`
}

type Detector interface {
	Check(metric string, series []Point) []Finding
}

// ZScore flags points beyond sigma standard deviations from the mean.
type ZScore struct {
	Sigma   float64
	MinData int
}

func (z ZScore) Check(metric string, series []Point) []Finding {
	if len(series) < z.MinData || z.MinData < 3 {
		return nil
	}
	var sum, sq float64
	for _, p := range series {
		sum += p.Value
		sq += p.Value * p.Value
	}
	n := float64(len(series))
	mean := sum / n
	variance := sq/n - mean*mean
	if variance <= 0 {
		return nil
	}
	std := math.Sqrt(variance)
	var out []Finding
	for _, p := range series {
		zs := (p.Value - mean) / std
		if math.Abs(zs) >= z.Sigma {
			out = append(out, Finding{Metric: metric, At: p.At, Value: p.Value, ZScore: zs, Reason: "zscore"})
		}
	}
	return out
}

// Threshold flags any point outside [low, high].
type Threshold struct {
	Low, High float64
}

func (t Threshold) Check(metric string, series []Point) []Finding {
	var out []Finding
	for _, p := range series {
		if p.Value < t.Low || p.Value > t.High {
			out = append(out, Finding{Metric: metric, At: p.At, Value: p.Value, Reason: "threshold"})
		}
	}
	return out
}
