module github.com/thanos-io/thanosbench

require (
	github.com/a8m/mark v0.1.1-0.20170507133748-44f2db618845 // indirect
	github.com/bwplotka/mimic v0.0.0-20190730202618-06ab9976e8ef
	github.com/gernest/wow v0.1.0 // indirect
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/analysis v0.19.4 // indirect
	github.com/go-openapi/runtime v0.19.3 // indirect
	github.com/go-openapi/swag v0.19.4
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.1-0.20191002090509-6af20e3a5340 // indirect
	github.com/mailru/easyjson v0.0.0-20190626092158-b2ccc519800e // indirect
	github.com/minio/cli v1.20.0 // indirect
	github.com/oklog/run v1.0.0
	github.com/oklog/ulid v1.3.1
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.7.0
	github.com/prometheus/prometheus v1.8.2-0.20191126064551-80ba03c67da1 // v1.8.2 is misleading as Prometheus does not have v2 module.
	github.com/thanos-io/thanos v0.9.0
	github.com/uber-go/atomic v1.4.0 // indirect
	go.mongodb.org/mongo-driver v1.0.4 // indirect
	go.uber.org/automaxprocs v1.2.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.25.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/api v0.0.0-20191115095533-47f6de673b26
	k8s.io/apimachinery v0.0.0-20191115015347-3c7067801da2
	k8s.io/client-go v12.0.0+incompatible // indirect
)

// We want to replace the client-go version with a specific commit hash,
// so that we don't get errors about being incompatible with the Go proxies.
// See https://github.com/thanos-io/thanos/issues/1415
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190620085554-14e95df34f1f
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/klog => k8s.io/klog v0.3.1
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
)

go 1.13
