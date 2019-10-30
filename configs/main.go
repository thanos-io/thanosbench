package main

import (
	"github.com/bwplotka/mimic"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	"github.com/thanos-io/thanosbench/configs/abstractions/secret"
	bench "github.com/thanos-io/thanosbench/configs/internal/benchmarks"
	"gopkg.in/alecthomas/kingpin.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
		// Resources for monitor observing benchmarks/tests.
		bench.GenMonitor(generator.With("monitor", "gen-manifests"), namespace)
		bench.GenCadvisor(generator.With("cadvisor", "gen-manifests"), namespace)
	}

	// Generate resources for various benchmarks.
	{
		generator := generator.With("remote-read", "gen-manifests")

		bench.GenRemoteReadBenchPrometheus(generator, "prometheus", namespace, dockerimage.PublicPrometheus("v2.12.0"), dockerimage.PublicThanos("v0.7.0"))
		bench.GenRemoteReadBenchPrometheus(generator, "prometheus-rr-streamed", namespace, dockerimage.PublicPrometheus("v2.13.0"), dockerimage.PublicThanos("v0.7.0"))
	}
	{
		generator := generator.With("lts", "gen-manifests")

		const storeAPILabelSelector = "lts-api"
		bench.GenThanosStoreGateway(generator, bench.StoreGatewayOpts{
			Name:      "store-base",
			Namespace: namespace,
			// Version with expected baseline memory usage.
			Img: dockerimage.PublicThanos("master-2019-09-03-f7a238fd"),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
			// Turned off cache for less moving parts.
			IndexCacheBytes:       "0MB",
			ChunkCacheBytes:       "2GB",
			StoreAPILabelSelector: storeAPILabelSelector,

			// You need secret for this.
			/*
				apiVersion: v1
				kind: Secret
				metadata:
				  name: s3
				data:
				  s3.yaml: |
				    <base64 config>
			*/
			ObjStoreSecret: secret.NewFile(
				"s3.yaml",
				"s3",
				"/s3/config",
			),
			ReadinessPath: "/metrics",
		})
		bench.GenThanosQuerier(generator, bench.QuerierOpts{
			Name:      "query-base",
			Namespace: namespace,
			Img:       dockerimage.PublicThanos("master-2019-10-29-b7f3ac9e"),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
			StoreAPILabelSelector: storeAPILabelSelector,
		})
		bench.GenThanosStoreGateway(generator, bench.StoreGatewayOpts{
			Name:      "store-test",
			Namespace: namespace,
			// Version introducing regression.
			Img: dockerimage.PublicThanos("master-2019-09-04-e8bf6f5d"),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
			IndexCacheBytes:       "0MB",
			ChunkCacheBytes:       "2GB",
			StoreAPILabelSelector: storeAPILabelSelector,
			ObjStoreSecret: secret.NewFile(
				"s3.yaml",
				"s3",
				"/s3/config",
			),
			ReadinessPath: "/metrics",
		})
	}
}
