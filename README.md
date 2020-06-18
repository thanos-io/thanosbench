# thanosbench

[![CircleCI](https://circleci.com/gh/thanos-io/thanosbench.svg?style=svg)](https://circleci.com/gh/thanos-io/thanosbench)
[![Go Report Card](https://goreportcard.com/badge/github.com/thanos-io/thanosbench)](https://goreportcard.com/report/github.com/thanos-io/thanosbench)
[![GoDoc](https://godoc.org/github.com/thanos-io/thanosbench?status.svg)](https://godoc.org/github.com/thanos-io/thanosbench)
[![Slack](https://img.shields.io/badge/join%20slack-%23thanos-brightgreen.svg)](https://slack.cncf.io/)

Kubernetes Playground for Thanos testing &amp; benchmarking purposes

## thanosbench

This repo adds additional tooling for benchmarks.

See `make build && ./thanosbench --help` for available commands or read below:

### WAL generation

[embedmd]:# (autogendocs/flags_walgen.txt)
```txt
usage: thanosbench walgen --output.dir=OUTPUT.DIR [<flags>]

Generates TSDB data into WAL files.

Flags:
  -h, --help                     Show context-sensitive help (also try
                                 --help-long and --help-man).
      --version                  Show application version.
      --log.level=info           Log filtering level.
      --log.format=logfmt        Log format to use.
      --config-file=<file-path>  Path to YAML for series config. See
                                 walgen.Config for the format.
      --config=<content>         Alternative to 'config-file' flag (lower
                                 priority). Content of YAML for series config.
                                 See walgen.Config for the format.
      --output.dir=OUTPUT.DIR    Output directory for generated TSDB data.

```

Config format:

[embedmd]:# (autogendocs/config_walgen.txt)
```txt
inputseries:
- type: ""
  characteristics:
    jitter: 0
    scrapeInterval: 0s
    changeInterval: 0s
    max: 0
    min: 0
  result:
    resulttype: 0
    result:
    - metric: {}
      value: 0
      timestamp: 0
  replicate: 0
retention: 0s
scrapeinterval: 0s
```

For example:

```yaml
inputseries:
- type: "gauge"
  characteristics:
    jitter: 20
    scrapeInterval: 15000000000
    changeInterval: 3600000000000
    max: 200000000
    min: 100000000
  result:
    resultType: "vector"
    result:
      - metric:
          __name__: "kube_pod_container_resource_limits_memory_bytes"
          cluster: "eu1"
          container: "addon-resizer"
          instance: "172.17.0.9:8080"
          job: "kube-state-metrics"
          namespace: "kube-system"
          node: "node1"
          pod: "kube-state-metrics-68f6cc566c-vp566"
        value: 1
        timestamp: 0
  replicate: 2
retention: 3600
scrapeinterval: 15

```

### Block plan & gen

[embedmd]:# (autogendocs/flags_block_plan.txt)
```txt
usage: thanosbench block plan --profile=PROFILE --labels=<name>="<value>" [<flags>]

Plan generates blocks specs used by blockgen command to build blocks.

Example plan with generation:

./thanosbench block plan -p <profile> --labels 'cluster="one"' --max-time
2019-10-18T00:00:00Z | ./thanosbench block gen --output.dir ./genblocks
--workers 20

Flags:
  -h, --help               Show context-sensitive help (also try --help-long and
                           --help-man).
      --version            Show application version.
      --log.level=info     Log filtering level.
      --log.format=logfmt  Log format to use.
  -p, --profile=PROFILE    Name of the harcoded profile to use
      --max-time=30m       If empty current time - 30m (usual consistency delay)
                           is used.
      --labels=<name>="<value>" ...
                           External labels for block stream (repeated).

```

Above outputs []blockgen.BlockSpec:

[embedmd]:# (autogendocs/config_blockspec.txt)
```txt
- meta:
    blockmeta:
      ulid: "00000000000000000000000000"
      mintime: 0
      maxtime: 0
      stats:
        numsamples: 0
        numseries: 0
        numchunks: 0
        numtombstones: 0
      compaction:
        level: 0
        sources: []
        deletable: false
        parents: []
        failed: false
      version: 0
    thanos:
      labels: {}
      downsample:
        resolution: 0
      source: ""
  series: []
```

Then block gen accepts this as input:

[embedmd]:# (autogendocs/flags_block_gen.txt)
```txt
usage: thanosbench block gen --output.dir=OUTPUT.DIR [<flags>]

Generates Prometheus/Thanos TSDB blocks from input. Expects []blockgen.BlockSpec
in YAML format as input.

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long
                               and --help-man).
      --version                Show application version.
      --log.level=info         Log filtering level.
      --log.format=logfmt      Log format to use.
      --output.dir=OUTPUT.DIR  Output directory for generated data.
      --workers=WORKERS        Number of go routines for block generation. If 0,
                               2*runtime.GOMAXPROCS(0) is used.

```

### Stress

[embedmd]:# (autogendocs/flags_stress.txt)
```txt
usage: thanosbench stress --workers=WORKERS [<flags>] <target>

Stress tests a remote StoreAPI.

Flags:
  -h, --help                  Show context-sensitive help (also try --help-long
                              and --help-man).
      --version               Show application version.
      --log.level=info        Log filtering level.
      --log.format=logfmt     Log format to use.
      --workers=WORKERS       Number of go routines for stress testing.
      --timeout=60s           Timeout of each operation
      --query.look-back=300h  How much time into the past at max we should look
                              back

Args:
  <target>  IP:PORT pair of the target to stress.

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
