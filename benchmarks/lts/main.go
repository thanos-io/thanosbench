package main

import (
	"regexp"

	"github.com/bwplotka/mimic"
	"github.com/thanos-io/thanosbench/benchmarks"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	"github.com/thanos-io/thanosbench/configs/abstractions/secret"
	k8s "github.com/thanos-io/thanosbench/configs/kubernetes"
	"gopkg.in/alecthomas/kingpin.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func thanosImageFromFlag(tag string) dockerimage.Image {
	if regexp.MustCompile(`^(v[0-9]+\.|master-).*`).MatchString(tag) {
		return dockerimage.PublicThanos(tag)
	}

	return dockerimage.Image{
		Organization: "",
		Project:      "thanos-local",
		Version:      tag,
	}
}

func main() {
	var tag *string
	generator := mimic.New(func(cmd *kingpin.CmdClause) {
		cmd.GetFlag("output").Default("benchmarks/lts/manifests")
		tag = cmd.Flag("tag", "Thanos docker image to use for deployment").Required().String()
	})

	// Make sure to generate at the very end.
	defer generator.Generate()

	{
		const storeAPILabelSelector = "lts-api-base"
		k8s.GenThanosStoreGateway(generator, k8s.StoreGatewayOpts{
			Name:      "store-base",
			Namespace: benchmarks.Namespace,
			Img:       dockerimage.PublicThanos("v0.12.2"),
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
			// NOTE(bwplotka): Turned off cache for less moving parts.
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
		})
		k8s.GenThanosQuerier(generator, k8s.QuerierOpts{
			Name:      "query-base",
			Namespace: benchmarks.Namespace,
			Img:       dockerimage.PublicThanos("v0.12.2"),
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
	}
	{
		const storeAPILabelSelector = "lts-api"
		k8s.GenThanosStoreGateway(generator, k8s.StoreGatewayOpts{
			Name:      "store",
			Namespace: benchmarks.Namespace,
			// Feel free to play with different versions!
			Img: thanosImageFromFlag(*tag),
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
			// NOTE(bwplotka): Turned off cache for less moving parts.
			IndexCacheBytes:       "0MB",
			ChunkCacheBytes:       "2GB",
			StoreAPILabelSelector: storeAPILabelSelector,
			ObjStoreSecret: secret.NewFile(
				"s3.yaml",
				"s3",
				"/s3/config",
			),
		})
		k8s.GenThanosQuerier(generator, k8s.QuerierOpts{
			Name:      "query",
			Namespace: benchmarks.Namespace,
			Img:       thanosImageFromFlag(*tag),
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
	}
}
