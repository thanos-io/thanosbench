# thanosbench

[![CircleCI](https://circleci.com/gh/thanos-io/thanosbench.svg?style=svg)](https://circleci.com/gh/thanos-io/thanosbench)
[![Go Report Card](https://goreportcard.com/badge/github.com/thanos-io/thanosbench)](https://goreportcard.com/report/github.com/thanos-io/thanosbench)
[![GoDoc](https://godoc.org/github.com/thanos-io/thanosbench?status.svg)](https://godoc.org/github.com/thanos-io/thanosbench)
[![Slack](https://img.shields.io/badge/join%20slack-%23thanos-brightgreen.svg)](https://slack.cncf.io/)

Kubernetes Playground for Thanos testing &amp; benchmarking purposes

## CLI

This repo adds additional tooling for benchmarks. See possible subcommands here:

See `make build && ./thanosbench --help` for available commands.

```
usage: thanosbench [<flags>] <command> [<args> ...]

Benchmarking tools for Thanos

Flags:
  -h, --help               Show context-sensitive help (also try --help-long and --help-man).
      --version            Show application version.
      --log.level=info     Log filtering level.
      --log.format=logfmt  Log format to use.

Commands:
  help [<command>...]
    Show help.

  walgen --output.dir=OUTPUT.DIR [<flags>]
    Generates TSDB data into WAL files.

  block gen --output.dir=OUTPUT.DIR [<flags>]
    Generates Prometheus/Thanos TSDB blocks from input. Expects []blockgen.BlockSpec in YAML format as input.

  block plan --profile=PROFILE --labels=<name>="<value>" [<flags>]
    Plan generates blocks specs used by blockgen command to build blocks.

    Example plan with generation:

    ./thanosbench block plan -p realistic-k8s-1w-small --labels 'cluster="one"' --max-time 2019-10-18T00:00:00Z | ./thanosbench block gen --output.dir ./genblocks --workers 20

  stress --workers=WORKERS [<flags>] <target>
    Stress tests a remote StoreAPI.
```

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

`make gen && kubectl apply -f benchmarks/monitor/manifests`

 For any adjustment, edit [benchmarks/monitor/main.go](https://github.com/thanos-io/thanosbench/blob/db8874ab23f480f33cdb4ac4eeec57562f566dd8/benchmarks/monitor/main.go#L25) or related template.
 `make gen` will generate the YAMLs.

Prometheus is configured to monitor only the namespace configured in `namespace` argument. With few pods it should took at most 100MB of memory on average.

3. Forward port to see Prometheus UI: `kubectl port-forward svc/monitor 9090:9090`

`kind` has cadvisor built in and default Prometheus is set to monitor it.

### Benchmarks

* [Remote read](benchmarks/remote-read/README.md)
* [Long term storage read path](benchmarks/lts/README.md)

## Potential next steps

* Mores sophisticated profiles for `block gen`.
* More benchmarks.
* Allow packing thanos and thanosbench binaries from certain commits into docker with ease (manual right now)
   * (?) framework for deploying manifests? As kubectl plugin?
