package bench

import (
	"time"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/providers/prometheus"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
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
		BlockgenImg: dockerimage.Image{Organization: "quay.io/thanos", Project: "thanosbench", Version: "docker-2019-10-04-19e823a"},
		// Generate 10k series of type gauge on start.
		BlockgenConfig: &blockgen.Config{
			InputSeries: []blockgen.Series{
				{
					Type:      "gauge",
					Jitter:    20,
					Max:       200000000,
					Min:       100000000,
					Replicate: 10000,
					Result: blockgen.QueryData{
						Result: model.Vector{
							{
								Metric: map[model.LabelName]model.LabelValue{
									"__name__":  "kube_pod_container_resource_limits_memory_bytes",
									"cluster":   "eu1",
									"container": "addon-resizer",
									"instance":  "172.17.0.9:8080",
									"job":       "kube-state-metrics",
									"namespace": "kube-system",
									"node":      "node1",
									"pod":       "kube-state-metrics-68f6cc566c-vp566",
								},
							},
						},
						ResultType: model.ValVector,
					},
				},
			},
			Retention: 10 * time.Hour,
		},

		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("5Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("5Gi"),
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
}
