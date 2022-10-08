package k8s

import (
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/thanos-io/thanos/pkg/block/metadata"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"github.com/thanos-io/thanosbench/pkg/seriesgen"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/providers/prometheus"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	"github.com/thanos-io/thanosbench/pkg/walgen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func GenRemoteReadBenchPrometheusWith10h10kSeriesWAL(gen *mimic.Generator, name string, namespace string, prometheusImg, thanosImg dockerimage.Image) {
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
		Retention:      "2d",
		ThanosbenchImg: dockerimage.Image{Organization: "quay.io/thanos", Project: "thanosbench", Version: "docker-2019-10-04-19e823a"},
		// Generate 10k series of type gauge on start.
		WalGenConfig: &walgen.Config{
			InputSeries: []walgen.Series{
				{
					Type: "gauge",
					Characteristics: seriesgen.Characteristics{
						Jitter:         20,
						Max:            200000000,
						Min:            100000000,
						ScrapeInterval: 15 * time.Second,
						ChangeInterval: 1 * time.Hour,
					},
					Replicate: 10000,
					Result: walgen.QueryData{
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

func GenRemoteReadBenchPrometheusWith1MoBlocks1kSeries(gen *mimic.Generator, name string, namespace string, prometheusImg, thanosImg dockerimage.Image) {
	maxTime, err := time.Parse(
		time.RFC3339,
		"2020-08-05T10:00:00+00:00")
	if err != nil {
		mimic.PanicErr(err)
	}

	seriesFn := func(mint, maxt time.Time) []blockgen.SeriesSpec {
		return []blockgen.SeriesSpec{
			{
				Labels:  labels.New(labels.Label{Name: "__name__", Value: "my_metric"}, labels.Label{Name: "a", Value: "1"}),
				Type:    blockgen.Gauge,
				MaxTime: timestamp.FromTime(maxt),
				MinTime: timestamp.FromTime(mint),
				Targets: 1000,
				Characteristics: seriesgen.Characteristics{
					Max:            200000000,
					Min:            10000000,
					Jitter:         30000000,
					ScrapeInterval: 15 * time.Second,
					ChangeInterval: 1 * time.Hour,
				},
			},
		}
	}

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
		Retention:      "999d",
		ThanosbenchImg: dockerimage.Image{Organization: "quay.io/thanos", Project: "thanosbench", Version: "chunk-iter-2020-08-05-4a32777"},
		// TODO(bwplotka): Use plan for that.
		BlockgenSpecs: []blockgen.BlockSpec{
			{
				Series: seriesFn(maxTime.Add(-2*time.Hour), maxTime),
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    timestamp.FromTime(maxTime),
						MinTime:    timestamp.FromTime(maxTime.Add(-2 * time.Hour)),
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
				},
			},
			{
				Series: seriesFn(maxTime.Add(-24*time.Hour), maxTime.Add(-2*time.Hour).Add(-1*time.Millisecond)),
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    timestamp.FromTime(maxTime.Add(-2 * time.Hour).Add(-1 * time.Millisecond)),
						MinTime:    timestamp.FromTime(maxTime.Add(-24 * time.Hour)),
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
				},
			},
			{
				Series: seriesFn(maxTime.Add(-2*24*time.Hour), maxTime.Add(-24*time.Hour).Add(-1*time.Millisecond)),
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    timestamp.FromTime(maxTime.Add(-24 * time.Hour).Add(-1 * time.Millisecond)),
						MinTime:    timestamp.FromTime(maxTime.Add(-2 * 24 * time.Hour)),
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
				},
			},
			{
				Series: seriesFn(maxTime.Add(-3*24*time.Hour), maxTime.Add(-2*24*time.Hour).Add(-1*time.Millisecond)),
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    timestamp.FromTime(maxTime.Add(-2 * 24 * time.Hour).Add(-1 * time.Millisecond)),
						MinTime:    timestamp.FromTime(maxTime.Add(-3 * 24 * time.Hour)),
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
				},
			},
			{
				Series: seriesFn(maxTime.Add(-7*24*time.Hour), maxTime.Add(-3*24*time.Hour).Add(-1*time.Millisecond)),
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    timestamp.FromTime(maxTime.Add(-3 * 24 * time.Hour).Add(-1 * time.Millisecond)),
						MinTime:    timestamp.FromTime(maxTime.Add(-7 * 24 * time.Hour)),
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
				},
			},
			{
				Series: seriesFn(maxTime.Add(-2*7*24*time.Hour), maxTime.Add(-7*24*time.Hour).Add(-1*time.Millisecond)),
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    timestamp.FromTime(maxTime.Add(-7 * 24 * time.Hour).Add(-1 * time.Millisecond)),
						MinTime:    timestamp.FromTime(maxTime.Add(-2 * 7 * 24 * time.Hour)),
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
				},
			},
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
