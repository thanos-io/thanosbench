package k8s

import (
	"fmt"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/encoding"
	"github.com/go-openapi/swag"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type QuerierOpts struct {
	Namespace string
	Name      string

	Img       dockerimage.Image
	Resources corev1.ResourceRequirements

	StoreAPILabelSelector string

	ReadinessPath string
	// TODO(bwplotka): Add static storeAPIs option.
}

func GenThanosQuerier(gen *mimic.Generator, opts QuerierOpts) {
	const (
		replicas = 1
		httpPort = 19190
		grpcPort = 19090
	)

	// Special headless k8s service for SRV lookup to discover local StoreAPIs.
	// https://kubernetes.io/docs/concepts/services-networking/service/#headless-services
	storeAPISrv := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name + "-store-apis",
			Namespace: opts.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				opts.StoreAPILabelSelector: "true",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       grpcPort,
					TargetPort: intstr.FromInt(grpcPort),
				},
			},
			ClusterIP: corev1.ClusterIPNone,
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

	querierContainer := corev1.Container{
		Name:    "thanos",
		Image:   opts.Img.String(),
		Command: []string{"thanos"},
		Args: []string{
			"query",
			"--log.level=debug",
			"--debug.name=$(POD_NAME)",
			// Large limits!
			fmt.Sprintf("--query.max-concurrent=%d", 99999999),
			fmt.Sprintf("--query.timeout=%s", "2h"),
			fmt.Sprintf("--query.replica-label=%s", "replica"),
			fmt.Sprintf("--http-address=0.0.0.0:%d", httpPort),
			fmt.Sprintf("--grpc-address=0.0.0.0:%d", grpcPort),
			fmt.Sprintf("--store=dnssrv+%s.default.svc.cluster.local:%d", storeAPISrv.Name, grpcPort),
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
			SuccessThreshold: 3,
			TimeoutSeconds:   10,
			FailureThreshold: 3,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/healthy",
					Port: intstr.FromInt(httpPort),
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
			{
				Name:          "grpc",
				ContainerPort: grpcPort,
			},
		},
		Resources: opts.Resources,
	}

	dpl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				selectorName: opts.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: swag.Int32(replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						selectorName: opts.Name,
					},
					Annotations: map[string]string{
						"version": opts.Img.String(),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{querierContainer},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectorName: opts.Name,
				},
			},
		},
	}
	gen.Add(opts.Name+".yaml", encoding.GhodssYAML(storeAPISrv, dpl, srv))
}
