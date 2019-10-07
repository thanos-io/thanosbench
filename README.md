# thanosbench

[![CircleCI](https://circleci.com/gh/thanos-io/thanosbench.svg?style=svg)](https://circleci.com/gh/thanos-io/thanosbench)
[![Go Report Card](https://goreportcard.com/badge/github.com/thanos-io/thanosbench)](https://goreportcard.com/report/github.com/thanos-io/thanosbench)
[![GoDoc](https://godoc.org/github.com/thanos-io/thanosbench?status.svg)](https://godoc.org/github.com/thanos-io/thanosbench)
[![Slack](https://img.shields.io/badge/join%20slack-%23thanos-brightgreen.svg)](https://slack.cncf.io/)

Kubernetes Playground for Thanos testing &amp; benchmarking purposes 

## Available tools

See `make build && ./thanosbench --help` for available commands.
 
## Repo structure:

* `cmds/thanosbench` - single binary for all tools.
* `config` - mimic-style Go configurations (e.g to deploy Thanos or Prometheus on opinionated Kubernetes)
* `pkg` - library of non-configuration Go packages. 
* `benchmarks` - set of benchmarks/tests for different cases/issue/testing aimed currently for kubernetes.
  * `<benchmark name>` - directory for benchmark. All is using [mimic](https://github.com/bwplotka/mimic) for the manifests generation. See [example](/benchmarks/remote-read)
    * gen-manifests - generated YAMLs. 
    * tests - directory for all test scripts (preferable in Go).
    * README.md
    
Use `make gen` to generate `config` templates into `benchmarks`.
    
## How to run benchmarks?

### Prerequisites

1. You need any recent Kubernetes. The easiest way is to run [`kind`](https://github.com/kubernetes-sigs/kind) however
bear in mind that most of the benchmarks are around memory allocations, so it's advised to perform tests on at least 16GB machine.

2. Before any benchmarks it is advised to start separate Prometheus instance which will measure results.

You can do on `default` namespace by running:

`make gen && kubectl apply -f benchmarks/monitor-gen-manifests/monitor-roles.yaml -f benchmarks/monitor-gen-manifests/monitor.yaml`    
    
 For any adjustment, edit [configs/main.go](https://github.com/thanos-io/thanosbench/blob/db8874ab23f480f33cdb4ac4eeec57562f566dd8/configs/main.go#L25) or related template. 
 `make gen` will generate the YAMLs.
 
Prometheus is configured to monitor only the namespace configured in `namespace` argument. With few pods it should took at most 100MB of memory on average. 
 
3. Forward port to see Prometheus UI: `kubectl port-forward svc/monitor 9090:9090`
 
4. (Optionally) if you run e.g on GKE, you might want to run your own `cadvisor` daemon set: 

`make gen && kubectl apply -f benchmarks/monitor-gen-manifests/cadvisor.yaml`  

`kind` has advisor built in and default Prometheus is set to monitor it.

### Benchmarks

* [Remote read](benchmarks/remote-read/README.md)
    
## Potential next steps

* Mores sophisticated features for `blockgen`.
* More benchmarks.
* Allow packing thanos and thanosbench binaries from certain commits into docker with ease (manual right now)
   * (?) framework for deploying manifests? As kubectl plugin?