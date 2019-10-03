package main

import (
	bench "github.com/thanos-io/thanosbench/configs"

	"github.com/bwplotka/mimic"
)

func main() {
	generator := mimic.New()

	// Make sure to generate at the very end.
	defer generator.Generate()

	// Generate resources for remote read tests.

	// Baseline.
	bench.GenRRTestPrometheus(
		generator,
		"prom-rr-test",
		"v2.11.0-rc.0-clear",
		"v0.5.0",
	)

	// Streamed.
	bench.GenRRTestPrometheus(
		generator,
		"prom-rr-test-streamed",
		"v2.11.0-rc.0-rr-streaming",
		"v0.5.0-rr-streamed2",
	)
}
