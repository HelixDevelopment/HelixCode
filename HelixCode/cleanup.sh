#!/bin/bash

# Clean up temporary files and build artifacts
echo "🧹 Cleaning up temporary files and logs..."

# Remove coverage files
rm -f *.out
rm -f coverage*.out
rm -f cov_*.out

# Remove compiled binaries
rm -f cli desktop aurora-os harmony-os main multi-agent-system performance-optimization-standalone

# Remove bin directory if it exists
rm -rf bin/

# Remove dist directory if it exists  
rm -rf dist/

echo "✅ Cleanup complete"