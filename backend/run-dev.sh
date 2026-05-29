#!/usr/bin/env bash
set -e

# Check if Air is installed
if ! command -v air &> /dev/null
then
    echo "Air is not installed. Installing it now..."
    go install github.com/air-verse/air@latest
    
    # Ensure GOPATH/bin is in PATH for this session if not already
    export PATH=$PATH:$(go env GOPATH)/bin
fi

echo "Starting backend with Air (Hot Reload)..."
air
