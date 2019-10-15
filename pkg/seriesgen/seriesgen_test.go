package seriesgen

import (
	"math/rand"
	"testing"
	"time"

	"github.com/thanos-io/thanos/pkg/testutil"
)

func TestCounterGen(t *testing.T) {
	g := NewCounterGen(rand.New(rand.NewSource(1)), 100, int64((24*time.Hour).Seconds())*1000, Characteristics{
		Jitter:         300,
		ScrapeInterval: 15 * time.Second,
		Min:            100,
		Max:            400,
	})

	lastV := float64(0)
	lastT := int64(0)

	init := false
	samples := int64(0)
	for g.Next() {
		samples++
		if init {
			ts, val := g.At()
			testutil.Assert(t, lastV <= val, "")
			testutil.Assert(t, lastT <= ts, "")
			init = true
		}
		lastT, lastV = g.At()
	}
	testutil.Equals(t, int64((24*time.Hour)/(15*time.Second)), samples)
}
