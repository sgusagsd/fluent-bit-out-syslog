#!/usr/bin/env bash
set -e
export GOFLAGS="-mod=vendor"
function log {
    local msg=$*
    date_timestamp=$(date +['%Y-%m-%d %H:%M:%S'])
    echo -ne "$date_timestamp $msg\\n"

    if [ -n "$LOG_FILE" ]; then
        echo -ne "$date_timestamp $msg\\n" >> "$LOG_FILE"
    fi
}

function install_golangci_lint {
        log "Install golangci-lint"
        curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b "$(go env GOPATH)/bin" v1.16.0
}

function run_golangci_lint {
    log "Run golangci-lint run"
    "$(go env GOPATH)/bin/golangci-lint" run
}

install_golangci_lint
run_golangci_lint
