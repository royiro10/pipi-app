#!/bin/bash

# Log function
log() {
    local LOG_LEVEL=$1
    shift
    local LOG_MSG="$@"
    local TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
    echo "${TIMESTAMP} [${LOG_LEVEL}] ${LOG_MSG}"
}

REPO_URL=https://github.com/royiro10/pipi-app
EXE_FILE="./pipi"

sudo apt update
log INFO "updated successfully"

log DEBUG "installing golang..."
sudo apt install -y golang
go version
log INGO "installed golang"

log INFO "build pipi"
make build

log INFO "run pipi"
GEMINAI_API_KEY=$GEMINAI_API_KEY $EXE_FILE
