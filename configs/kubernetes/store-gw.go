package k8s

import (
	"fmt"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/abstractions/kubernetes/volumes"
	"github.com/bwplotka/mimic/encoding"
	"github.com/go-openapi/swag"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	"github.com/thanos-io/thanosbench/configs/abstractions/secret"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type StoreGatewayOpts struct {
	Namespace string
	Name      string

	Img       dockerimage.Image
	Resources corev1.ResourceRequirements

	IndexCacheBytes string // Defaults: 250MB
	ChunkCacheBytes string // Defaults: 2GB

	StoreAPILabelSelector string

	ObjStoreSecret secret.File

	ReadinessPath string
}

// NOTE: No persistent volume on purpose to simplify testing. It is must-have on the production setup.
func GenThanosStoreGateway(gen *mimic.Generator, opts StoreGatewayOpts) {
	const (
		replicas = 1
		dataPath = "/data"

		httpPort = 19190
		grpcPort = 19090
	)

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
			Type: corev1.ServiceTypeClusterIP,
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
					Name:       "grpc",
					Port:       grpcPort,
					TargetPort: intstr.FromInt(grpcPort),
				},
			},
		},
	}

	sharedVM := volumes.VolumeAndMount{
		VolumeMount: corev1.VolumeMount{
			Name:      opts.Name,
			MountPath: dataPath,
		},
	}

	storeContainer := corev1.Container{
		Name:    "thanos",
		Image:   opts.Img.String(),
		Command: []string{"thanos"},
		Args: []string{
			"store",
			"--log.level=debug",
			"--debug.name=$(POD_NAME)",
			fmt.Sprintf("--objstore.config-file=%s", opts.ObjStoreSecret.Path()),
			fmt.Sprintf("--index-cache-size=%s", opts.IndexCacheBytes),
			fmt.Sprintf("--chunk-pool-size=%s", opts.ChunkCacheBytes),
			fmt.Sprintf("--http-address=0.0.0.0:%d", httpPort),
			fmt.Sprintf("--grpc-address=0.0.0.0:%d", grpcPort),
			fmt.Sprintf("--data-dir=%s", dataPath),
		},
		Env: []corev1.EnvVar{
			// NOTE: Add following env var for old go memory management: {Name: "GODEBUG", Value:"madvdontneed=1"}.
			{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			}},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt(httpPort),
					Path: func() string {
						if opts.ReadinessPath == "" {
							return "/-/ready"
						}
						return opts.ReadinessPath
					}(),
				},
			},
			InitialDelaySeconds: 120,
			SuccessThreshold:    3,
			TimeoutSeconds:      10,
			FailureThreshold:    3,
		},
		//LivenessProbe: &corev1.Probe{
		//	ProbeHandler: corev1.ProbeHandler{
		//		HTTPGet: &corev1.HTTPGetAction{
		//			Path: "/-/healthy",
		//			Port: intstr.FromInt(httpPort),
		//		},
		//	},
		//	InitialDelaySeconds: 350,
		//	TimeoutSeconds:      30,
		//	FailureThreshold:    3,
		//},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: httpPort,
			},
			{
				Name:          "grpc",
				ContainerPort: grpcPort,
			},
		},
		VolumeMounts: volumes.VolumesAndMounts{sharedVM, opts.ObjStoreSecret.VolumeAndMount}.VolumeMounts(),
		Resources:    opts.Resources,
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
					Annotations: map[string]string{
						"version": opts.Img.String(),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{storeContainer},
					Volumes:    volumes.VolumesAndMounts{sharedVM, opts.ObjStoreSecret.VolumeAndMount}.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectorName: opts.Name,
				},
			},
		},
	}
	gen.Add(opts.Name+".yaml", encoding.GhodssYAML(set, srv))
}
