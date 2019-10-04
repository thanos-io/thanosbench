package bench

import (
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"

	"github.com/prometheus/common/model"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/providers/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func GenRemoteReadBenchPrometheus(gen *mimic.Generator, name string, namespace string, prometheusImg, thanosImg dockerimage.Image) {
	GenPrometheus(gen, PrometheusOpts{
		Namespace: namespace,
		Name:      name,

		Img:       prometheusImg,
		ThanosImg: thanosImg,

		// Empty config.
		Config: prometheus.Config{
			GlobalConfig: prometheus.GlobalConfig{
				ExternalLabels: map[model.LabelName]model.LabelValue{
					"replica": "0",
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("10Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("10Gi"),
			},
		},
		ThanosResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("5Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("5Gi"),
			},
		},

		DisableCompactions: true,
	})

	/*
		blockgenImage   = "improbable/blockgen:master-894c9481c4"
				// Generate 10k series.
				blockgenInput = `[{
		  "type": "gauge",
		  "jitter": 20,
		  "max": 200000000,
		  "min": 100000000,
		  "result": {"multiplier":10000,"resultType":"vector","result":[{"metric":{"__name__":"kube_pod_container_resource_limits_memory_bytes","cluster":"eu1","container":"addon-resizer","instance":"172.17.0.9:8080","job":"kube-state-metrics","namespace":"kube-system","node":"minikube","pod":"kube-state-metrics-68f6cc566c-vp566"}}]}
		}]`
	*/
}
