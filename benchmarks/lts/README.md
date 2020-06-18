# LTS

This definition contains various Store GW statefulsets definitions with Querier deployment
set up to benchmark long term retention.

It serves an example setup for benchmarking Thanos long term retention.
It might be used for future for automated per PR test.

For now it is used for adhoc hacking and benchmarking.

## Requirements

* See [README.md](/README.md#Prerequisites) (example of setting up cluster + monitor)
* Object storage bucket:
  * Prepare Thanos YAML objstore configuration: [see this doc](https://thanos.io/storage.md/#configuration)
  * Add the yaml as a secret to your K8s cluster
  * Change accordingly the [go definitions](main.go) and run `make gen` or `go run benchmarks/lts/main.go generate --tag=<thanos quay image tag>`

## Usage

* Generate YAMLs from definitions:
  * `make gen` or `go run benchmarks/lts/main.go generate --tag=<thanos quay image tag>` any time to regenerate the output YAMLs.

You should see [2 Store GW and 2 Queries](/benchmarks/lts/manifests).

One Path can be used as a baseline, second can be used to test differences across versions.

* Generate test dataset

To have proper benchmark we have stable dataset. To generate it you can use `thanosbench block gen` for example:

```
./thanosbench block plan -p realistic-k8s-1w-small --labels 'cluster="one"' --labels 'dataset="realistic"' --max-time 2019-10-18T00:00:00Z | \
  ./thanosbench block gen --output.dir genblocks/test --workers 20

./thanosbench block plan -p continuous-1w-small --labels 'cluster="one"' --labels 'dataset="continuous"' --max-time 2019-10-18T00:00:00Z | \
  ./thanosbench block gen --output.dir genblocks/test --workers 20
```

NOTE: This roughly requires 8GB of memory to finish.

Upload to object storage using:

```
./thanos-replicate run --one-off --objstoreto.config-file=<your objstore YAML> --objstorefrom.config="
type: FILESYSTEM
config:
  directory: genblocks/test

"
```

* Apply any definition you want to run:
  * `kubectl apply -f benchmarks/lts/manifests/<choose>.yaml`

Dataset mentioned above use roughly 12MB baseline memory.

* Forward Querier port:
  * `kubectl port-forward pod/$(kubectl get po | grep query | cut -f1 -d " ") 19190:19190`

* Use Querier to query long term storage. Note that for above dataset you need to query before `2019-10-18T00:00:00Z`:

For example:

http://localhost:19190/graph?g0.range_input=2d&g0.end_input=2019-10-18%2000%3A00&g0.max_source_resolution=0s&g0.expr=k8s_app_metric0&g0.tab=0

This fetches 5k series over few days. This for current version takes around 120MB of memory on Store GW.

* See resource consumption based on requests (assuming your `monitor` Prometheus is port forwarded on localhost:9090)

### Resources:

http://localhost:9090/graph?g0.range_input=1h&g0.expr=sum(container_memory_working_set_bytes%7Bpod%3D~%22store.*%7Cquery.*%22%2C%20container!%3D%22%22%7D)%20by%20(pod%2C%20container)&g0.tab=0&g1.range_input=1h&g1.expr=go_memstats_alloc_bytes%7Bpod%3D~%22store.*%7Cquery.*%22%7D&g1.tab=0&g2.range_input=1h&g2.expr=%20sum(rate(container_cpu_user_seconds_total%7Bpod%3D~%22store.*%7Cquery.*%22%2C%20container!%3D%22%22%7D%5B5m%5D))%20by%20(pod%2C%20container)&g2.tab=0&g3.range_input=1h&g3.expr=go_memstats_mspan_inuse_bytes%7Bpod%3D~%22store.*%7Cquery.*%22%7D&g3.tab=0&g4.range_input=1h&g4.expr=sum(container_memory_mapped_file%7Bpod%3D~%22store.*%7Cquery.*%22%2C%20container!%3D%22%22%7D)%20by%20(pod%2C%20container)%20&g4.tab=0

### Block/Series

http://localhost:9090/graph?g0.range_input=1h&g0.expr=sum(increase(thanos_objstore_bucket_operations_total%5B5m%5D))%20by%20(operation%2C%20instance%2C%20pod)%20&g0.tab=0&g1.range_input=30m&g1.expr=(sum(increase(thanos_bucket_store_series_data_size_fetched_bytes_sum%5B5m%5D)%20%2F%0Aincrease(thanos_bucket_store_series_data_size_fetched_bytes_count%5B5m%5D))%20without%20(data_type))%20%2F%20increase(thanos_bucket_store_series_blocks_queried_sum%5B5m%5D)&g1.tab=0&g2.range_input=30m&g2.expr=increase(thanos_bucket_store_series_data_size_fetched_bytes_sum%5B5m%5D)%20%2F%0Aincrease(thanos_bucket_store_series_data_size_fetched_bytes_count%5B5m%5D)&g2.tab=0&g3.range_input=1h&g3.expr=increase(thanos_bucket_store_series_data_size_fetched_bytes_sum%5B5m%5D)&g3.tab=0&g4.range_input=30m&g4.expr=histogram_quantile(0.95%2C%20sum(rate(thanos_objstore_bucket_operation_duration_seconds_bucket%5B1m%5D))%20by%20(operation%2C%20instance%2C%20pod%2C%20le))&g4.tab=0&g5.range_input=1h&g5.expr=increase(thanos_bucket_store_series_blocks_queried_sum%5B5m%5D)&g5.tab=0

### gRPC

http://localhost:9090/graph?g0.range_input=2h&g0.expr=increase(grpc_server_msg_received_total%7Bgrpc_method%3D%22Series%22%7D%5B5m%5D)&g0.tab=0&g1.range_input=2h&g1.expr=increase(grpc_server_msg_sent_total%7Bgrpc_method%3D%22Series%22%7D%5B5m%5D)%20%2F%20increase(grpc_server_msg_received_total%7Bgrpc_method%3D%22Series%22%7D%5B5m%5D)&g1.tab=0
