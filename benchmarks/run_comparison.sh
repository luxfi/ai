#!/bin/bash
# C-Chain Block Building Benchmark: BadgerDB+gRPC vs ZapDB+ZAP
# Copyright (C) 2025, Lux Industries Inc.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="${SCRIPT_DIR}/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "${RESULTS_DIR}"

echo "=========================================="
echo "C-Chain Block Building Benchmark"
echo "Comparing: BadgerDB+gRPC vs ZapDB+ZAP"
echo "Timestamp: ${TIMESTAMP}"
echo "=========================================="
echo ""

# Build benchmarks
echo "Building benchmarks..."
cd "${SCRIPT_DIR}"
go test -c -o benchmark.test . 2>/dev/null || {
    echo "Creating go.mod for benchmarks..."
    cat > go.mod << 'EOF'
module github.com/luxfi/ai/benchmarks

go 1.25.5

require (
    github.com/luxfi/database v1.17.42
    github.com/luxfi/ids v1.2.9
    github.com/luxfi/metric v1.5.0
)
EOF
    go mod tidy
    go test -c -o benchmark.test .
}

echo ""
echo "Running ZapDB benchmarks..."
echo "=========================================="

# Run ZapDB write benchmarks
./benchmark.test -test.bench="BenchmarkZapDBBlockWrite" -test.benchmem -test.benchtime=5s 2>&1 | tee "${RESULTS_DIR}/zapdb_write_${TIMESTAMP}.txt"

echo ""
echo "Running ZapDB read benchmarks..."
./benchmark.test -test.bench="BenchmarkZapDBBlockRead" -test.benchmem -test.benchtime=5s 2>&1 | tee "${RESULTS_DIR}/zapdb_read_${TIMESTAMP}.txt"

echo ""
echo "Running ZapDB batch benchmarks..."
./benchmark.test -test.bench="BenchmarkZapDBBatchWrite" -test.benchmem -test.benchtime=5s 2>&1 | tee "${RESULTS_DIR}/zapdb_batch_${TIMESTAMP}.txt"

echo ""
echo "Running iterator benchmarks..."
./benchmark.test -test.bench="BenchmarkIteratorPerformance" -test.benchmem -test.benchtime=5s 2>&1 | tee "${RESULTS_DIR}/zapdb_iterator_${TIMESTAMP}.txt"

echo ""
echo "Running MemDB baseline benchmarks..."
./benchmark.test -test.bench="BenchmarkMemDBBaseline" -test.benchmem -test.benchtime=5s 2>&1 | tee "${RESULTS_DIR}/memdb_baseline_${TIMESTAMP}.txt"

echo ""
echo "=========================================="
echo "Benchmark Complete!"
echo "Results saved to: ${RESULTS_DIR}"
echo "=========================================="

# Generate summary
echo ""
echo "Summary:"
echo "--------"
cat "${RESULTS_DIR}"/*_${TIMESTAMP}.txt | grep -E "^Benchmark" | sort
