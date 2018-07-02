#!/usr/bin/env bash
set -e

function log() {
    local msg=$*
    date_timestamp=$(date +['%Y-%m-%d %H:%M:%S'])
    echo -ne "$date_timestamp $msg\n"
}

function install_golangci_lint () {
    if ! command -v golangci-lint 1>/dev/null 2>&1; then
        echo "Install golangci-lint"
        curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b $GOPATH/bin v1.8.1
    fi
}

function run_golangci_lint() {
    go get "github.com/onsi/ginkgo"
    go get "github.com/onsi/gomega"
    go get "github.com/oratos/out_syslog/pkg/fluentbin"
    go get "code.cloudfoundry.org/rfc5424"
    log "Run golangci-lint run"
    golangci-lint run
}

install_golangci_lint
run_golangci_lint
