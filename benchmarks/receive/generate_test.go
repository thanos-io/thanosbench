package main

import (
	"math/rand"
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"
)

func TestGenerateTimeSeries(t *testing.T) {
	rand.Seed(42)

	actual := generateTimeseries("test", []configTimeseries{{
		Name: "up",
		Labels: []configLabels{{
			Name:        "job",
			Cardinality: 2,
		}, {
			Name:        "instance",
			Cardinality: 3,
		}},
	}})

	expected := prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{
			{Labels: []prompb.Label{{Name: "__name__", Value: "up"}, {Name: "client", Value: "test"}, {Name: "job", Value: "hrukpt"}, {Name: "instance", Value: "neuvun"}}},
			{Labels: []prompb.Label{{Name: "__name__", Value: "up"}, {Name: "client", Value: "test"}, {Name: "job", Value: "tuezpt"}, {Name: "instance", Value: "huksqv"}}},
			{Labels: []prompb.Label{{Name: "__name__", Value: "up"}, {Name: "client", Value: "test"}, {Name: "job", Value: "hrukpt"}, {Name: "instance", Value: "gzadxl"}}},
			{Labels: []prompb.Label{{Name: "__name__", Value: "up"}, {Name: "client", Value: "test"}, {Name: "job", Value: "tuezpt"}, {Name: "instance", Value: "neuvun"}}},
			{Labels: []prompb.Label{{Name: "__name__", Value: "up"}, {Name: "client", Value: "test"}, {Name: "job", Value: "hrukpt"}, {Name: "instance", Value: "huksqv"}}},
			{Labels: []prompb.Label{{Name: "__name__", Value: "up"}, {Name: "client", Value: "test"}, {Name: "job", Value: "tuezpt"}, {Name: "instance", Value: "gzadxl"}}},
		},
	}

	require.Len(t, actual.Timeseries, 6)
	require.Equal(t, expected, actual)
}
