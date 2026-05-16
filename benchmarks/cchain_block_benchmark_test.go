// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/luxfi/database"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/database/zapdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/metric"
)

// BenchmarkConfig holds configuration for database benchmarks
type BenchmarkConfig struct {
	Name       string
	DBType     string
	NumBlocks  int
	BlockSize  int // bytes per block
	NumTxPerBlock int
}

// BlockData simulates C-chain block data
type BlockData struct {
	ID        ids.ID
	Height    uint64
	Timestamp int64
	TxCount   int
	Data      []byte
}

// generateBlockData creates test block data
func generateBlockData(height uint64, size int, txCount int) *BlockData {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	return &BlockData{
		ID:        ids.GenerateTestID(),
		Height:    height,
		Timestamp: time.Now().UnixNano(),
		TxCount:   txCount,
		Data:      data,
	}
}

// serializeBlock converts block to bytes (simplified)
func serializeBlock(block *BlockData) []byte {
	// In real implementation, this would use proper serialization
	result := make([]byte, 8+8+8+4+len(block.Data))
	// Copy height, timestamp, txcount, data
	copy(result[24:], block.Data)
	return result
}

// BenchmarkZapDBBlockWrite benchmarks writing blocks to ZapDB
func BenchmarkZapDBBlockWrite(b *testing.B) {
	configs := []BenchmarkConfig{
		{"SmallBlocks", "zapdb", 1000, 1024, 10},           // 1KB blocks, 10 tx
		{"MediumBlocks", "zapdb", 1000, 32768, 100},        // 32KB blocks, 100 tx
		{"LargeBlocks", "zapdb", 1000, 131072, 500},        // 128KB blocks, 500 tx
		{"XLargeBlocks", "zapdb", 500, 524288, 1000},       // 512KB blocks, 1000 tx
	}

	for _, cfg := range configs {
		b.Run(cfg.Name, func(b *testing.B) {
			// Create temp directory for database
			tmpDir := b.TempDir()

			// Create ZapDB
			db, err := zapdb.New(tmpDir, nil, "benchmark", metric.NewRegistry())
			if err != nil {
				b.Fatalf("Failed to create zapdb: %v", err)
			}
			defer db.Close()

			// Pre-generate blocks
			blocks := make([]*BlockData, cfg.NumBlocks)
			for i := 0; i < cfg.NumBlocks; i++ {
				blocks[i] = generateBlockData(uint64(i), cfg.BlockSize, cfg.NumTxPerBlock)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, block := range blocks {
					key := block.ID[:]
					value := serializeBlock(block)
					if err := db.Put(key, value); err != nil {
						b.Fatalf("Put failed: %v", err)
					}
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(cfg.NumBlocks*b.N)/b.Elapsed().Seconds(), "blocks/sec")
		})
	}
}

// BenchmarkZapDBBlockRead benchmarks reading blocks from ZapDB
func BenchmarkZapDBBlockRead(b *testing.B) {
	configs := []BenchmarkConfig{
		{"SmallBlocks", "zapdb", 1000, 1024, 10},
		{"MediumBlocks", "zapdb", 1000, 32768, 100},
		{"LargeBlocks", "zapdb", 1000, 131072, 500},
	}

	for _, cfg := range configs {
		b.Run(cfg.Name, func(b *testing.B) {
			tmpDir := b.TempDir()

			db, err := zapdb.New(tmpDir, nil, "benchmark", metric.NewRegistry())
			if err != nil {
				b.Fatalf("Failed to create zapdb: %v", err)
			}
			defer db.Close()

			// Pre-populate database
			blocks := make([]*BlockData, cfg.NumBlocks)
			for i := 0; i < cfg.NumBlocks; i++ {
				blocks[i] = generateBlockData(uint64(i), cfg.BlockSize, cfg.NumTxPerBlock)
				if err := db.Put(blocks[i].ID[:], serializeBlock(blocks[i])); err != nil {
					b.Fatalf("Setup put failed: %v", err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, block := range blocks {
					if _, err := db.Get(block.ID[:]); err != nil {
						b.Fatalf("Get failed: %v", err)
					}
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(cfg.NumBlocks*b.N)/b.Elapsed().Seconds(), "reads/sec")
		})
	}
}

// BenchmarkZapDBBatchWrite benchmarks batch writing (simulating block commits)
func BenchmarkZapDBBatchWrite(b *testing.B) {
	configs := []BenchmarkConfig{
		{"BatchSmall", "zapdb", 100, 1024, 10},
		{"BatchMedium", "zapdb", 100, 32768, 100},
		{"BatchLarge", "zapdb", 100, 131072, 500},
	}

	for _, cfg := range configs {
		b.Run(cfg.Name, func(b *testing.B) {
			tmpDir := b.TempDir()

			db, err := zapdb.New(tmpDir, nil, "benchmark", metric.NewRegistry())
			if err != nil {
				b.Fatalf("Failed to create zapdb: %v", err)
			}
			defer db.Close()

			// Pre-generate blocks
			blocks := make([]*BlockData, cfg.NumBlocks)
			for i := 0; i < cfg.NumBlocks; i++ {
				blocks[i] = generateBlockData(uint64(i), cfg.BlockSize, cfg.NumTxPerBlock)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				batch := db.NewBatch()
				for _, block := range blocks {
					if err := batch.Put(block.ID[:], serializeBlock(block)); err != nil {
						b.Fatalf("Batch put failed: %v", err)
					}
				}
				if err := batch.Write(); err != nil {
					b.Fatalf("Batch write failed: %v", err)
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(cfg.NumBlocks*b.N)/b.Elapsed().Seconds(), "blocks/sec")
		})
	}
}

// BenchmarkMemDBBaseline provides a baseline comparison using in-memory database
func BenchmarkMemDBBaseline(b *testing.B) {
	configs := []BenchmarkConfig{
		{"SmallBlocks", "memdb", 1000, 1024, 10},
		{"MediumBlocks", "memdb", 1000, 32768, 100},
		{"LargeBlocks", "memdb", 1000, 131072, 500},
	}

	for _, cfg := range configs {
		b.Run(cfg.Name, func(b *testing.B) {
			db := memdb.New()
			defer db.Close()

			blocks := make([]*BlockData, cfg.NumBlocks)
			for i := 0; i < cfg.NumBlocks; i++ {
				blocks[i] = generateBlockData(uint64(i), cfg.BlockSize, cfg.NumTxPerBlock)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, block := range blocks {
					if err := db.Put(block.ID[:], serializeBlock(block)); err != nil {
						b.Fatalf("Put failed: %v", err)
					}
				}
			}

			b.StopTimer()
			b.ReportMetric(float64(cfg.NumBlocks*b.N)/b.Elapsed().Seconds(), "blocks/sec")
		})
	}
}

// BenchmarkIteratorPerformance benchmarks iterator (used for state sync)
func BenchmarkIteratorPerformance(b *testing.B) {
	tmpDir := b.TempDir()

	db, err := zapdb.New(tmpDir, nil, "benchmark", metric.NewRegistry())
	if err != nil {
		b.Fatalf("Failed to create zapdb: %v", err)
	}
	defer db.Close()

	// Populate with 10k entries
	numEntries := 10000
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key-%08d", i)
		value := make([]byte, 256)
		if err := db.Put([]byte(key), value); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		iter := db.NewIterator()
		count := 0
		for iter.Next() {
			_ = iter.Key()
			_ = iter.Value()
			count++
		}
		iter.Release()

		if count != numEntries {
			b.Fatalf("Iterator count mismatch: got %d, want %d", count, numEntries)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(numEntries*b.N)/b.Elapsed().Seconds(), "entries/sec")
}

// SimulateCChainBlockProduction simulates realistic C-chain block production
func SimulateCChainBlockProduction(ctx context.Context, db database.Database, numBlocks int, txPerBlock int) (time.Duration, error) {
	start := time.Now()

	for i := 0; i < numBlocks; i++ {
		// Simulate block production
		block := generateBlockData(uint64(i), 32*1024, txPerBlock) // 32KB average block

		// Write block data
		batch := db.NewBatch()

		// Block header
		if err := batch.Put([]byte(fmt.Sprintf("block:header:%d", i)), block.Data[:256]); err != nil {
			return 0, err
		}

		// Block body
		if err := batch.Put([]byte(fmt.Sprintf("block:body:%d", i)), block.Data); err != nil {
			return 0, err
		}

		// Simulate transaction receipts
		for j := 0; j < txPerBlock; j++ {
			receipt := make([]byte, 256) // Average receipt size
			if err := batch.Put([]byte(fmt.Sprintf("tx:receipt:%d:%d", i, j)), receipt); err != nil {
				return 0, err
			}
		}

		// Commit block
		if err := batch.Write(); err != nil {
			return 0, err
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return time.Since(start), ctx.Err()
		default:
		}
	}

	return time.Since(start), nil
}
