package secret

import (
	"path"

	"github.com/bwplotka/mimic/abstractions/kubernetes/volumes"
	corev1 "k8s.io/api/core/v1"
)

type File struct {
	volumes.VolumeAndMount

	filePath string
}

func (f File) Path() string {
	return path.Join(f.MountPath, f.filePath)
}

func NewFile(filePath string, secretName string, mountPath string) File {
	return File{
		VolumeAndMount: VolumeAndMount(secretName, mountPath),
		filePath:       filePath,
	}
}

// VolumeAndMount creates a volume named after the secret and mounts it read-only at the given path.
// Note that this assumes/enforces:
// - The volume and secret will be named the same
// - All items in the secret will be mounted
func VolumeAndMount(secretName string, mountPath string) volumes.VolumeAndMount {
	return volumes.VolumeAndMount{
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
		VolumeMount: corev1.VolumeMount{
			Name:      secretName,
			MountPath: mountPath,
			ReadOnly:  true,
		},
	}
}
