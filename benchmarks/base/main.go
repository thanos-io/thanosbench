package main

import (
	"github.com/bwplotka/mimic"
	"github.com/thanos-io/thanosbench/benchmarks"
	k8s "github.com/thanos-io/thanosbench/configs/kubernetes"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	generator := mimic.New(func(cmd *kingpin.CmdClause) {
		cmd.GetFlag("output").Default("benchmarks/base/manifests")
	})

	// Make sure to generate at the very end.
	defer generator.Generate()

	// Resources for monitor observing benchmarks/tests.
	k8s.GenMonitor(generator, benchmarks.Namespace)
}

