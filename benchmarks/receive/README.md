# thanos-receive benchmark

This benchmark was set up so that we have a proper way of testing auto-scaling Thanos Receivers.

So far I've only run the tests locally with kind on a pretty beefy machine (AMD Ryzen 3900X) -
you might want to tweak some replicas counts in the `run.sh` before giving it a go on your machine.

Most of the logic of scaling up and down and deleting some running pods while at full load is within all `run.sh`.

## Getting started

1. `kind create cluster` to have s local cluster (you can skip if you have some other cluster available).
1. `kubectl create namespace thanos` create the necessary Thanos namespace.
1. Clone kube-prometheus and run `kubectl apply -f ./manifests/setup/` and `kubectl apply -f ./manifests/` from its root.
1. `kubectl delete alertmanagers.monitoring.coreos.com -n monitoring main` you can optionally delete alertmanagers.
1. `kubectl edit prometheuses.monitoring.coreos.com -n monitoring k8s` and edit the replicas to 1 for a simpler life during development.
1. Back in this repository run `kubectl apply -f ./benchmarks/receive/manifests/prometheus-operator` to configure the Prometheus to scrape out benchmark.
1. `kubectl apply -f ./benchmarks/receive/manifests/` to deploy the entire rest: Thanos Qurier, Thanos Receiver, Thanos Receive Router, Thanos Receive Controller and one instance of the custom Thanos Receive Benchmark.
1. In another terminal run `kubectl port-forward -n monitoring svc/grafana 3000` and log in to Grafana with `admin:admin`.
1. Upload the `ThanosReceiveBenchmark.json` dashboard.
1. Finally, run the benchmark with `./benchmarks/receive/run.sh`.

### Running another benchmark

1. Downscale all benchmark so there's no more traffic with `kubectl scale deployment -n thanos thanos-receive-benchmark --replicas 0`.
1. Wait until all Deployments & StatefulSets have minimum replica count (probably 3).
1. Delete all Receiver and Receive Router Pods with `kubectl delete pod -n thanos -l app.kubernetes.io/name=thanos-receive` && `kubectl delete pod -n thanos -l app.kubernetes.io/name=thanos-receive-route`.
1. Delete the Prometheus to start with fresh metrics `kubectl delete pod -n monitoring prometheus-k8s-0`.
1. Run the benchmark again with `./benchmarks/receive/run.sh`.
