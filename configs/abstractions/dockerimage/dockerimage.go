package dockerimage

import "fmt"

type Image struct {
	Organization string
	Project      string
	Version      string
}

func (i Image) String() string {
	prefix := i.Organization
	if prefix != "" {
		prefix += "/"
	}
	return fmt.Sprintf("%s%s:%s", prefix, i.Project, i.Version)
}

func PublicThanos(tag string) Image {
	return Image{
		Organization: "quay.io/thanos",
		Project:      "thanos",
		Version:      tag,
	}
}

func PublicPrometheus(tag string) Image {
	return Image{
		Organization: "quay.io/prometheus",
		Project:      "prometheus",
		Version:      tag,
	}
}
