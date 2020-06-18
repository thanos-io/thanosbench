# Remote Read

This definition contains Prometheus statefulsets definitions set up to test remote read changes described [here](https://docs.google.com/document/d/1JqrU3NjM9HoGLSTPYOvR217f5HBKBiJTqikEB9UiJL0/edit#).

[@bwplotka](https://bwplotka.dev/) is using those to benchmark new remote read with Thanos sidecar on live Kubernetes cluster.

## Requirements

* See [README.md](/README.md#Prerequisites)
* [grpcurl](https://github.com/fullstorydev/grpcurl)
* [pv](https://linux.die.net/man/1/pv)

## Usage

* Generate YAMLs from definitions:
  * `make gen`

You should see [2 Prometheus + Thanos definitions](/benchmarks/remote-read/manifests). One is baseline,
second is a version with modified remote read that allows streaming encoded chunks.

Those resources are crafted for benchmark purposes -> they generate artificial metric data in the init container.

* Apply baseline or improved version:
  * `kubectl apply -f benchmarks/remote-read/manifests/<choose>.yaml`

NOTE: because of init container generating data - init can take few minutes and lots of memory (roughly 6GB per 10k series).

* Forward gRPC sidecar port:
  * `kubectl port-forward pod/<pod name> 1234:19090`

* Perform tests using test.sh (modifying parameters in script itself - heavy queries!)
  * This performs heavy queries (query all) against Thanos gRPC Store.Series of sidecar which will proxy
requests as remote read to Prometheus
  * `bash ./benchmarks/remote-read/test/test.sh localhost:1234`

* See resource consumption based on requests (assuming your `monitor` Prometheus is running on localhost:9090)

http://localhost:9090/graph?g0.range_input=1h&g0.expr=sum(container_memory_working_set_bytes%7Bpod%3D~%22prometheus.*%22%2C%20container!%3D%22%22%7D)%20by%20(container)&g0.tab=0&g1.range_input=1h&g1.expr=go_memstats_alloc_bytes%7Bpod%3D~%22prometheus.*%22%7D&g1.tab=0&g2.range_input=1h&g2.expr=sum(rate(container_cpu_user_seconds_total%7Bpod%3D~%22prometheus.*%22%2C%20container!%3D%22%22%7D%5B5m%5D))%20by%20(container)&g2.tab=0
