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

		bench.GenRemoteReadBenchPrometheus(generator, "prometheus", namespace, dockerimage.PublicPrometheus("v2.12.0"), dockerimage.PublicThanos("v0.7.0"))
		bench.GenRemoteReadBenchPrometheus(generator, "prometheus-rr-streamed", namespace, dockerimage.PublicPrometheus("v2.13.0"), dockerimage.PublicThanos("v0.7.0"))
	}
}
