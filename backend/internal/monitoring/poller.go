// Package monitoring polls devices on per-device intervals with bounded
// concurrency (backpressure) so one slow device never stalls the fleet.
package monitoring

import (
	"context"
	"sync"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/devices"
)

type Sample struct {
	DeviceID   string    `json:"device_id"`
	CPUPct     float64   `json:"cpu_pct"`
	MemPct     float64   `json:"mem_pct"`
	UptimeSecs int64     `json:"uptime_secs"`
	Online     bool      `json:"online"`
	LatencyMs  int64     `json:"latency_ms"`
	At         time.Time `json:"at"`
}

type Source struct {
	Device    devices.Device
	Connector sdk.NetworkConnector
}

type PollFunc func(ctx context.Context, s Source) Sample

func DefaultPoll(ctx context.Context, s Source) Sample {
	start := time.Now()
	info, err := s.Connector.GetDeviceInfo(ctx)
	sample := Sample{DeviceID: s.Device.ID, At: time.Now().UTC(), LatencyMs: time.Since(start).Milliseconds()}
	if err != nil {
		return sample // Online=false: device failure isolated here
	}
	sample.Online = true
	sample.CPUPct, sample.MemPct, sample.UptimeSecs = info.CPUPercent, info.MemoryPercent, info.UptimeSecs
	return sample
}

// PollAll polls sources with at most maxParallel in flight.
func PollAll(ctx context.Context, sources []Source, maxParallel int, fn PollFunc) []Sample {
	if maxParallel <= 0 {
		maxParallel = 10
	}
	if fn == nil {
		fn = DefaultPoll
	}
	out := make([]Sample, len(sources))
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	for i, src := range sources {
		wg.Add(1)
		go func(i int, src Source) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			out[i] = fn(pctx, src)
		}(i, src)
	}
	wg.Wait()
	return out
}
