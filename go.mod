module github.com/thanos-io/thanosbench

require (
	github.com/bwplotka/mimic v0.0.0-20190730202618-06ab9976e8ef
	github.com/cespare/xxhash/v2 v2.1.1
	github.com/fatih/structtag v1.1.0
	github.com/go-kit/kit v0.10.0
	github.com/go-openapi/swag v0.19.9
	github.com/oklog/run v1.1.0
	github.com/oklog/ulid v1.3.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.11.1
	github.com/prometheus/prometheus v1.8.2-0.20200811193703-869f1bc587e6
	github.com/thanos-io/thanos v0.15.0
	go.uber.org/automaxprocs v1.2.0
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	google.golang.org/grpc v1.30.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
)

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.7.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/rodaine/hclencoder v0.0.0-20190213202847-fb9757bb536e // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/goleak v1.1.0 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20220919091848-fb04ddd9f9c8 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/tools v0.0.0-20200725200936-102e7d357031 // indirect
	google.golang.org/genproto v0.0.0-20200724131911-43cab4749ae7 // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog v1.0.0 // indirect
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

go 1.18
