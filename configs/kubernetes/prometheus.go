package k8s

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"time"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/abstractions/kubernetes/volumes"
	"github.com/bwplotka/mimic/encoding"
	"github.com/bwplotka/mimic/providers/prometheus"
	sdconfig "github.com/bwplotka/mimic/providers/prometheus/discovery/config"
	"github.com/bwplotka/mimic/providers/prometheus/discovery/kubernetes"
	"github.com/go-openapi/swag"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"github.com/thanos-io/thanosbench/pkg/walgen"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GenMonitor(gen *mimic.Generator, namespace string) {
	const name = "monitor"
	GenPrometheus(gen, PrometheusOpts{
		Namespace: namespace,
		Name:      name,

		Img:       dockerimage.PublicPrometheus("v2.13.0-rc.0"),
		ThanosImg: dockerimage.PublicThanos("v0.7.0"),
		Retention: "2d",
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		ThanosResources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
		},
		ServiceAccountName: name,
		Config: prometheus.Config{
			GlobalConfig: prometheus.GlobalConfig{
				ExternalLabels: map[model.LabelName]model.LabelValue{
					"monitor": "0",
				},
				ScrapeInterval: model.Duration(15 * time.Second),
			},
			ScrapeConfigs: []*prometheus.ScrapeConfig{
				{
					JobName: "kubernetes-nodes-cadvisor",
					ServiceDiscoveryConfig: sdconfig.ServiceDiscoveryConfig{
						KubernetesSDConfigs: []*kubernetes.SDConfig{{Role: kubernetes.RoleNode}},
					},
					RelabelConfigs: []*prometheus.RelabelConfig{
						{
							Action: prometheus.RelabelLabelMap,
							Regex:  prometheus.MustNewRegexp("__meta_kubernetes_node_label_(.+)"),
						},
						{
							Replacement: "kubernetes.default.svc:443",
							TargetLabel: "__address__",
						},
						{
							Regex:       prometheus.MustNewRegexp("(.+)"),
							Replacement: "/api/v1/nodes/${1}/proxy/metrics/cadvisor",
							TargetLabel: "__metrics_path__",
							SourceLabels: model.LabelNames{
								"__meta_kubernetes_node_name",
							},
						},
					},
					Scheme: "https",
					HTTPClientConfig: config.HTTPClientConfig{
						TLSConfig: config.TLSConfig{
							CAFile:             "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
							InsecureSkipVerify: true,
						},
						BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
				{
					JobName: "kubernetes-pods",
					ServiceDiscoveryConfig: sdconfig.ServiceDiscoveryConfig{
						KubernetesSDConfigs: []*kubernetes.SDConfig{
							{
								Role:               kubernetes.RolePod,
								NamespaceDiscovery: kubernetes.NamespaceDiscovery{Names: []string{namespace}},
							},
						},
					},
					RelabelConfigs: []*prometheus.RelabelConfig{
						{
							SourceLabels: model.LabelNames{"__meta_kubernetes_pod_container_port_name"},
							Action:       prometheus.RelabelKeep,
							Regex:        prometheus.MustNewRegexp("^(http|m-.+)$"),
							TargetLabel:  "__address__",
						},
						{
							SourceLabels: model.LabelNames{"__meta_kubernetes_pod_annotation_thanosbench_io_metric_path"},
							Action:       prometheus.RelabelReplace,
							Regex:        prometheus.MustNewRegexp("(.+)"),
							TargetLabel:  "__metrics_path__",
						},
						{
							SourceLabels: model.LabelNames{"__meta_kubernetes_namespace"},
							Action:       prometheus.RelabelReplace,
							TargetLabel:  "namespace",
						},
						{
							SourceLabels: model.LabelNames{"__meta_kubernetes_pod_name"},
							Action:       prometheus.RelabelReplace,
							TargetLabel:  "pod",
						},
						{
							SourceLabels: model.LabelNames{"__meta_kubernetes_pod_label_app"},
							Action:       prometheus.RelabelReplace,
							TargetLabel:  "job",
						},
						{
							SourceLabels: model.LabelNames{"__meta_kubernetes_pod_annotation_version"},
							Action:       prometheus.RelabelReplace,
							TargetLabel:  "version",
						},
						{
							Action: prometheus.RelabelReplace,
							SourceLabels: model.LabelNames{
								"job",
								"__meta_kubernetes_pod_container_port_name",
							},
							Regex:       prometheus.MustNewRegexp("(.+);m-(.+)"),
							Replacement: "$1-$2",
							TargetLabel: "job",
						},
					},
				},
			},
		},
	})

	// TODO(bwplotka): Consider scoping down to just `role` as we are fine with limiting montoring to one namespace.
	clr := v1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				selectorName: name,
			},
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{
					"nodes",
					"nodes/proxy",
					"services",
					"endpoints",
					"pods",
					"ingresses",
				},
				Verbs: []string{
					"list",
					"watch",
					"get",
				},
			},
			{
				NonResourceURLs: []string{"/metrics"},
				Verbs:           []string{"get"},
			},
		},
	}

	clrBinding := v1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				selectorName: name,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []v1.Subject{
			{
				Kind:      v1.ServiceAccountKind,
				Name:      name,
				Namespace: namespace,
			},
		},
	}

	svc := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				selectorName: name,
			},
		},
	}

	gen.Add(name+"-roles.yaml", encoding.GhodssYAML(clrBinding, clr, svc))
}

func genInPlace(r io.Reader) []byte {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		mimic.PanicErr(err)
	}
	return b
}

type PrometheusOpts struct {
	Namespace string
	Name      string

	Config    prometheus.Config
	Img       dockerimage.Image
	Resources corev1.ResourceRequirements

	// If empty, no data autogeneration will be defined.
	WalGenConfig   *walgen.Config
	BlockgenSpecs  []blockgen.BlockSpec
	ThanosbenchImg dockerimage.Image

	ThanosImg       dockerimage.Image
	ThanosResources corev1.ResourceRequirements

	DisableCompactions bool
	ServiceAccountName string
	Retention          string

	StoreAPILabelSelector string
}

// NOTE: No persistent volume on purpose to simplify testing. It is must-have!
func GenPrometheus(gen *mimic.Generator, opts PrometheusOpts) {
	const (
		replicas = 1

		configVolumeMount = "/etc/prometheus"
		sharedDataPath    = "/data-shared"

		httpPort        = 9090
		httpSidecarPort = 19190
		grpcSidecarPort = 19090
	)
	var (
		configVolumeName = fmt.Sprintf("%s-config", opts.Name)
		promDataPath     = path.Join(sharedDataPath, "prometheus")
	)

	promConfigAndMount := volumes.ConfigAndMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configVolumeName,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				selectorName: opts.Name,
			},
		},
		VolumeMount: corev1.VolumeMount{
			Name:      configVolumeName,
			MountPath: configVolumeMount,
		},
		Data: map[string]string{
			"prometheus.yaml": string(genInPlace(encoding.YAML(opts.Config))),
		},
	}

	sharedVM := volumes.VolumeAndMount{
		VolumeMount: corev1.VolumeMount{
			Name:      opts.Name,
			MountPath: sharedDataPath,
		},
	}

	srv := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				selectorName: opts.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Selector: map[string]string{
				selectorName: opts.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       httpPort,
					TargetPort: intstr.FromInt(httpPort),
				},
				{
					Name:       "grpc-sidecar",
					Port:       grpcSidecarPort,
					TargetPort: intstr.FromInt(grpcSidecarPort),
				},
				{
					Name:       "http-sidecar",
					Port:       httpSidecarPort,
					TargetPort: intstr.FromInt(httpSidecarPort),
				},
			},
		},
	}

	prometheusContainer := corev1.Container{
		Name:  "prometheus",
		Image: opts.Img.String(),
		Args: []string{
			fmt.Sprintf("--config.file=%v/prometheus.yaml", configVolumeMount),
			"--log.level=info",
			// Unlimited RR, useful for tests.
			"--storage.remote.read-concurrent-limit=99999",
			"--storage.remote.read-sample-limit=9999999999999999",
			fmt.Sprintf("--storage.tsdb.path=%s", promDataPath),
			"--storage.tsdb.min-block-duration=2h",
			// Avoid compaction for less moving parts in results.
			func() string {
				if opts.DisableCompactions {
					return "--storage.tsdb.max-block-duration=2h"
				}
				return "--storage.tsdb.max-block-duration=4h"
			}(),
			fmt.Sprintf("--storage.tsdb.retention.time=%s", opts.Retention),
			"--web.enable-lifecycle",
			"--web.enable-admin-api",
		},
		Env: []corev1.EnvVar{
			// NOTE: Add following env var for old go memory management: {Name: "GODEBUG", Value:"madvdontneed=1"}.
			{Name: "HOSTNAME", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			}},
		},
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt(int(httpPort)),
					Path: "-/ready",
				},
			},
			SuccessThreshold: 3,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/healthy",
					Port: intstr.FromInt(9090),
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      30,
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: httpPort,
			},
		},
		VolumeMounts: volumes.VolumesAndMounts{promConfigAndMount.VolumeAndMount(), sharedVM}.VolumeMounts(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: swag.Bool(false),
			RunAsUser:    swag.Int64(1000),
		},
		Resources: opts.Resources,
	}

	// TODO(bwplotka): Allow dynamic rule/config reload.
	thanosSidecarContainer := corev1.Container{
		Name:            "thanos",
		Image:           opts.ThanosImg.String(),
		Command:         []string{"thanos"},
		ImagePullPolicy: corev1.PullAlways,
		Args: []string{
			"sidecar",
			"--log.level=debug",
			"--debug.name=$(POD_NAME)",
			fmt.Sprintf("--http-address=0.0.0.0:%d", httpSidecarPort),
			fmt.Sprintf("--grpc-address=0.0.0.0:%d", grpcSidecarPort),
			fmt.Sprintf("--prometheus.url=http://localhost:%d", httpPort),
			fmt.Sprintf("--tsdb.path=%s", promDataPath),
		},
		Env: []corev1.EnvVar{
			// NOTE: Add following env var for old go memory management: {Name: "GODEBUG", Value:"madvdontneed=1"}.
			{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			}},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "m-sidecar",
				ContainerPort: httpSidecarPort,
			},
			{
				Name:          "grpc-sidecar",
				ContainerPort: grpcSidecarPort,
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt(int(httpSidecarPort)),
					Path: "metrics",
				},
			},
		},
		VolumeMounts: volumes.VolumesAndMounts{sharedVM}.VolumeMounts(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: swag.Bool(false),
			RunAsUser:    swag.Int64(1000),
		},
		Resources: opts.ThanosResources,
	}

	var initContainers []corev1.Container
	if opts.WalGenConfig != nil {
		initContainers = append(initContainers, corev1.Container{
			Name:    "walgen",
			Image:   opts.ThanosbenchImg.String(),
			Command: []string{"/bin/thanosbench"},
			Args: []string{
				"walgen",
				fmt.Sprintf("--config=%s", string(genInPlace(encoding.YAML(*opts.WalGenConfig)))),
				fmt.Sprintf("--output.dir=%s", promDataPath),
			},
			VolumeMounts: []corev1.VolumeMount{sharedVM.VolumeMount},
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
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: swag.Bool(false),
				RunAsUser:    swag.Int64(1000),
			},
		})
	}
	if len(opts.BlockgenSpecs) > 0 {
		initContainers = append(initContainers, corev1.Container{
			Name:    "blockgen",
			Image:   opts.ThanosbenchImg.String(),
			Command: []string{"/bin/thanosbench"},
			Args: []string{
				"block", "gen",
				fmt.Sprintf("--config=%s", string(genInPlace(encoding.YAML(opts.BlockgenSpecs)))),
				fmt.Sprintf("--output.dir=%s", promDataPath),
			},
			VolumeMounts: []corev1.VolumeMount{sharedVM.VolumeMount},
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
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: swag.Bool(false),
				RunAsUser:    swag.Int64(1000),
			},
		})
	}

	set := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				selectorName: opts.Name,
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    swag.Int32(replicas),
			ServiceName: opts.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: func() map[string]string {
						if opts.StoreAPILabelSelector == "" {
							return map[string]string{selectorName: opts.Name}
						}
						return map[string]string{
							selectorName:               opts.Name,
							opts.StoreAPILabelSelector: "true",
						}
					}(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: opts.ServiceAccountName,
					InitContainers:     initContainers,
					Containers:         []corev1.Container{prometheusContainer, thanosSidecarContainer},
					Volumes:            volumes.VolumesAndMounts{promConfigAndMount.VolumeAndMount(), sharedVM}.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectorName: opts.Name,
				},
			},
		},
	}
	gen.Add(opts.Name+".yaml", encoding.GhodssYAML(set, srv, promConfigAndMount.ConfigMap()))
}
