# Prometheus definitions for remote read.

This definition contains Prometheus statefulsets definitions set up to test remote read changes described [here](https://docs.google.com/document/d/1JqrU3NjM9HoGLSTPYOvR217f5HBKBiJTqikEB9UiJL0/edit#).

## Usage

Generate Kubernetes YAML:

`go run benchmarks/prometheus-remote-read/main.go generate -o benchmarks/prometheus-remote-read/gen-manifests`

See the `benchmarks/prometheus-remote-read/manifests` directory.

Those are 2 Prometheus + Thanos. One is baseline, second is a version with modified remote read that allows streaming encoded chunks.
You can use `kubectl apply` to deploy those. 

Those resources are crafted for benchmark purposes -> they generate artificial metric data.

[@bwplotka](https://bwplotka.dev/) is using those to benchmark new remote read with Thanos sidecar on live Kubernetes cluster.  

### Running test.

* Generate YAMLs from definitions:
  * `go run benchmarks/prometheus-remote-read/main.go generate -o benchmarks/prometheus-remote-read/gen-manifests`

* Apply baseline:
  * `kubectl apply -f benchmarks/prometheus-remote-read/gen-manifests/prom-rr-test.yaml`

* Forward gRPC sidecar port:
  * `kubectl port-forward pod/prom-rr-test-0 1234:19090`

* Perform tests using test.sh (modifying parameters in script itself - heavy queries!)
  * This performs heavy queries against Thanos gRPC Store.Series of sidecar which will proxy
requests as remote read to Prometheus
  * `bash ./benchmarks/prometheus-remote-read/test/test.sh localhost:1234`
