package main

import (
	"math/rand"

	"github.com/prometheus/prometheus/prompb"
)

func generateTimeseries(client string, timeseries []configTimeseries) prompb.WriteRequest {
	wreq := prompb.WriteRequest{}

	for _, ts := range timeseries {
		labelValues := make([][]string, 0, len(ts.Labels))

		cardinality := 1
		for _, label := range ts.Labels {
			cardinality *= label.Cardinality

			vs := make([]string, label.Cardinality)
			for j := 0; j < label.Cardinality; j++ {
				vs[j] = generateString(6)
			}
			labelValues = append(labelValues, vs)
		}

		wTimeseries := make([]prompb.TimeSeries, cardinality)
		for i := 0; i < cardinality; i++ {
			wTimeseries[i] = prompb.TimeSeries{
				Labels: make([]prompb.Label, len(ts.Labels)+2),
			}
			wTimeseries[i].Labels[0] = prompb.Label{Name: "__name__", Value: ts.Name}
			wTimeseries[i].Labels[1] = prompb.Label{Name: "client", Value: client}
			for j, label := range ts.Labels {
				wTimeseries[i].Labels[j+2] = prompb.Label{
					Name:  label.Name,
					Value: labelValues[j][i%label.Cardinality],
				}
			}
		}
		wreq.Timeseries = wTimeseries
	}

	return wreq
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func generateString(length uint) string {
	s := make([]rune, length)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
