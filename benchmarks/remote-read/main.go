package main

import (
	"github.com/bwplotka/mimic"
	"github.com/thanos-io/thanosbench/benchmarks"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	k8s "github.com/thanos-io/thanosbench/configs/kubernetes"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	generator := mimic.New(func(cmd *kingpin.CmdClause) {
		cmd.GetFlag("output").Default("benchmarks/remote-read/manifests")
	})

	// Make sure to generate at the very end.
	defer generator.Generate()

	k8s.GenRemoteReadBenchPrometheus(generator, "prometheus", benchmarks.Namespace, dockerimage.PublicPrometheus("v2.17.2"), dockerimage.PublicThanos("v0.12.1"))
}
