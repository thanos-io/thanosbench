package k8s

import (
	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/abstractions/kubernetes/volumes"
	"github.com/bwplotka/mimic/encoding"
	"github.com/go-openapi/swag"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenCadvisor(gen *mimic.Generator, namespace string) {
	const (
		name     = "cadvisor"
		httpPort = 8080
	)

	caVolumes := volumes.VolumesAndMounts{
		{
			VolumeMount: corev1.VolumeMount{
				Name:      "rootfs",
				MountPath: "/rootfs",
				ReadOnly:  true,
			},
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
		{
			VolumeMount: corev1.VolumeMount{
				Name:      "var-run",
				MountPath: "/var/run",
				ReadOnly:  true,
			},
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run",
				},
			},
		},
		{
			VolumeMount: corev1.VolumeMount{
				Name:      "sys",
				MountPath: "/sys",
				ReadOnly:  true,
			},
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "sys",
				},
			},
		},
		{
			VolumeMount: corev1.VolumeMount{
				Name:      "docker",
				MountPath: "/var/lib/docker",
				ReadOnly:  true,
			},
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/docker",
				},
			},
		},
		{
			VolumeMount: corev1.VolumeMount{
				Name:      "disk",
				MountPath: "/dev/disk",
				ReadOnly:  true,
			},
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/dev/disk",
				},
			},
		},
	}

	cadvisor := corev1.Container{
		Name:  "cadvisor",
		Image: dockerimage.Image{Organization: "registry.k8s.io", Project: "cadvisor", Version: "v0.30.2"}.String(),
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: httpPort,
			},
		},
		VolumeMounts: caVolumes.VolumeMounts(),
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("150m"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("300m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
	}

	daemon := appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				selectorName: name,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						selectorName: name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            name,
					Containers:                    []corev1.Container{cadvisor},
					TerminationGracePeriodSeconds: swag.Int64(30),
					Volumes:                       caVolumes.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					selectorName: name,
				},
			},
		},
	}

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
				APIGroups:     []string{"policy"},
				Resources:     []string{"podsecuritypolicies"},
				Verbs:         []string{"use"},
				ResourceNames: []string{name},
			},
		},
	}

	podSecPolicy := v1beta1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodSecurityPolicy",
			APIVersion: "policy/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				selectorName: name,
			},
		},
		Spec: v1beta1.PodSecurityPolicySpec{
			Volumes:            []v1beta1.FSType{"*"},
			SELinux:            v1beta1.SELinuxStrategyOptions{Rule: v1beta1.SELinuxStrategyRunAsAny},
			RunAsUser:          v1beta1.RunAsUserStrategyOptions{Rule: v1beta1.RunAsUserStrategyRunAsAny},
			SupplementalGroups: v1beta1.SupplementalGroupsStrategyOptions{Rule: v1beta1.SupplementalGroupsStrategyRunAsAny},
			FSGroup:            v1beta1.FSGroupStrategyOptions{Rule: v1beta1.FSGroupStrategyRunAsAny},
			AllowedHostPaths: []v1beta1.AllowedHostPath{
				{PathPrefix: "/"},
				{PathPrefix: "/var/run"},
				{PathPrefix: "/sys"},
				{PathPrefix: "/var/lib/docker"},
				{PathPrefix: "/dev/disk"},
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

	gen.Add(name+".yaml", encoding.GhodssYAML(podSecPolicy, clr, clrBinding, svc, daemon))
}
