#!/bin/bash

echo "Testing build environment..."

# Check Go version
go version

# Check if we're in the right directory
pwd
ls -la

echo "Test complete"