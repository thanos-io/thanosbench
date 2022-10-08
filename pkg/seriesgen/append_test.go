package seriesgen

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/storage"
	"github.com/thanos-io/thanos/pkg/testutil"
)

type testSet struct {
	count int
	curr  Series
}

func (s *testSet) Next() bool {
	if s.count <= 0 {
		return false
	}
	s.count--

	s.curr = NewSeriesGen(labels.Labels{
		{
			Name:  "a",
			Value: fmt.Sprintf("%v", s.count),
		},
	},
		NewValGen(rand.New(rand.NewSource(int64(s.count))), 0, timestamp.FromTime(time.Unix(0, 0).Add(40*time.Second)), Characteristics{
			ScrapeInterval: 10 * time.Second,
			Max:            100,
			Min:            200,
		}),
	)
	return true
}

func (s *testSet) At() Series { return s.curr }

func (s *testSet) Err() error { return nil }

type testAppendable struct {
	mtx     sync.Mutex
	samples map[uint64][]sample
}

func (a *testAppendable) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	hash := uint64(ref)
	if hash == 0 {
		hash = l.Hash()
	}
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.samples[hash] = append(a.samples[hash], sample{T: t, V: v})
	return storage.SeriesRef(hash), nil
}

func (a *testAppendable) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	return 0, nil
}

func (a *testAppendable) UpdateMetadata(ref storage.SeriesRef, l labels.Labels, m metadata.Metadata) (storage.SeriesRef, error) {
	return 0, nil
}

func (a *testAppendable) Commit() error {
	return nil
}

func (a *testAppendable) Rollback() error {
	return nil
}

func (a *testAppendable) Appender(_ context.Context) storage.Appender {
	return a
}

func TestAppend(t *testing.T) {
	s := &testSet{
		count: 2,
	}

	a := &testAppendable{samples: map[uint64][]sample{}}
	testutil.Ok(t, Append(context.Background(), 2*runtime.GOMAXPROCS(0), a, s))
	testutil.Equals(t, map[uint64][]sample{
		0x6577cd4df75e4415: {
			{T: 10000, V: 140.13863149001767},
			{T: 20000, V: 106.88960028354377},
			{T: 30000, V: 134.20855473136945},
			{T: 40000, V: 156.66629546848895},
			{T: 50000, V: 157.9608877899447},
		},
		0xc552620224fd8b78: {
			{T: 10000, V: 106.42558121988247},
			{T: 20000, V: 175.7484565559158},
			{T: 30000, V: 135.06032974565488},
			{T: 40000, V: 194.61995987962965},
			{T: 50000, V: 163.6088665433866},
		},
	}, a.samples)

}
