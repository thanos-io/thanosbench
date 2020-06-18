#!/usr/bin/env bash
#
# Generate --help output for all commands and embed them into the component docs.
set -u
set +x

EMBEDMD_BIN=${EMBEDMD_BIN:-embedmd}
SED_BIN=${SED_BIN:-sed}
THANOSBENCH_BIN=${THANOSBENCH_BIN:-${GOBIN}/thanosbench}

function docs {
# If check arg was passed, instead of the docs generation verifies if docs coincide with the codebase.
if [[ "${CHECK}" == "check" ]]; then
    set +e
    DIFF=$(${EMBEDMD_BIN} -d *.md)
    RESULT=$?
    if [[ "$RESULT" != "0" ]]; then
        cat << EOF
Docs have discrepancies, do 'make docs' and commit changes:

${DIFF}
EOF
        exit 2
    fi
else
    ${EMBEDMD_BIN} -w *.md
fi

}

if ! [[ "$0" =~ "scripts/genflagdocs.sh" ]]; then
	echo "must be run from repository root"
	exit 255
fi

CHECK=${1:-}

# Auto update flags.
mkdir -p autogendocs

commands=("walgen" "stress")
for x in "${commands[@]}"; do
    ${THANOSBENCH_BIN} "${x}" --help &> "autogendocs/flags_${x}.txt"
done

blockCommands=("gen" "plan")
for x in "${blockCommands[@]}"; do
    ${THANOSBENCH_BIN} block "${x}" --help &> "autogendocs/flags_block_${x}.txt"
done

# Remove white noise.
${SED_BIN} -i -e 's/[ \t]*$//' autogendocs/*.txt

# Auto update configuration.
go run scripts/cfggen/main.go --output-dir=autogendocs/

# Embed generated things!
docs
