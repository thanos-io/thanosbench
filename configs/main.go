package main

import (
	"github.com/bwplotka/mimic"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	bench "github.com/thanos-io/thanosbench/configs/internal/benchmarks"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "default"
)

func main() {
	generator := mimic.New(func(cmd *kingpin.CmdClause) {
		cmd.GetFlag("output").Default("benchmarks")
	})

	// Make sure to generate at the very end.
	defer generator.Generate()

	{
		generator := generator.With("monitor-gen-manifests")
		// Generate resources for common monitor for our benchmarks: Minimal Prometheus.
		bench.GenMonitor(generator, namespace)
		bench.GenCadvisor(generator, namespace)
	}

	// Generate resources for various benchmarks.
	{
		generator := generator.With("remote-read", "gen-manifests")

		bench.GenRemoteReadBenchPrometheus(generator, "prometheus", namespace, dockerimage.Image{}, dockerimage.Image{})
		bench.GenRemoteReadBenchPrometheus(generator, "prometheus-rr-streamed", namespace, dockerimage.Image{}, dockerimage.Image{})

		//// Baseline.
		//bench.GenRRTestPrometheus(
		//	generator,
		//	,
		//	"v2.11.0-rc.0-clear",
		//	"v0.5.0",
		//)
		//
		//// Streamed.
		//bench.GenRRTestPrometheus(
		//	generator,
		//	"prom-rr-test-streamed",
		//	"v2.11.0-rc.0-rr-streaming",
		//	"v0.5.0-rr-streamed2",
		//)
	}
}
