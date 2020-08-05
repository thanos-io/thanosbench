#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd ${DIR}/

TARGET_URL=$1

MIN_TDIFF="-10 hours"
MAX_TDIFF="-2 hours"

MIN_T=$(($(date +%s%N -d "${MIN_TDIFF}")/1000000))
MAX_T=$(($(date +%s%N -d "${MAX_TDIFF}")/1000000))

date
echo "Fetching 10k series StoreAPI.Series from sidecar via RR"
echo "'min_time'=${MIN_T} ${MIN_TDIFF} 'max_time'=${MAX_T} ${MAX_TDIFF}"
GOGOPROTO_ROOT="$(GO111MODULE=on go list -f '{{ .Dir }}' -m github.com/gogo/protobuf)"

time "$(go env GOBIN)/grpcurl" \
-plaintext -proto \
./rpc.proto -proto ./types.proto \
-import-path . \
-import-path ${GOGOPROTO_ROOT} \
-import-path ${GOGOPROTO_ROOT}/protobuf \
-d @ \
${TARGET_URL} thanos.Store/Series <<EOM | pv -b >/dev/null
{
  "minTime": ${MIN_T},
  "maxTime": ${MAX_T},
  "matchers": [{
    "type": 2,
    "name": "blockgen_fake_replica",
    "value": "^\d{1,4}$"
  }]
}
EOM


# Take half:  "value": "^[1-4]?\d{1,3}$"
# Take all:   "value": "^\d{1,4}$"

###
#  int64 min_time                 = 1;
#  int64 max_time                 = 2;
#  repeated LabelMatcher matchers = 3 [(gogoproto.nullable) = false];
#
#  int64 max_resolution_window = 4;
#  repeated Aggr aggregates    = 5;
#
#  // Deprecated. Use partial_response_strategy instead.
#  bool partial_response_disabled = 6;
###

###
#// Matcher specifies a rule, which can match or set of labels or not.
#  enum Type {
#    EQ  = 0; // =
#    NEQ = 1; // !=
#    RE  = 2; // =~
#    NRE = 3; // !~
#  }
#  Type type    = 1;
#  string name  = 2;
#  string value = 3;
###