module github.com/thanos-io/thanosbench

require (
	github.com/bwplotka/mimic v0.0.0-20190730202618-06ab9976e8ef
	github.com/go-kit/kit v0.10.0
	github.com/go-openapi/swag v0.19.9
	github.com/oklog/run v1.1.0
	github.com/oklog/ulid v1.3.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.10.0
	github.com/prometheus/prometheus v1.8.2-0.20200110114423-1e64d757f711
	github.com/thanos-io/thanos v0.11.0
	go.uber.org/automaxprocs v1.2.0
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	google.golang.org/grpc v1.29.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
)

// We want to replace the client-go version with a specific commit hash,
// so that we don't get errors about being incompatible with the Go proxies.
// See https://github.com/thanos-io/thanos/issues/1415
replace (
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20200609165731-66dfb951c4ca
	k8s.io/api => k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190620085554-14e95df34f1f
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/klog => k8s.io/klog v0.3.1
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
)

go 1.13
