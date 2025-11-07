// Package benchmarks provides performance benchmarks for HelixCode
package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// BenchmarkTaskCreation measures task creation performance
func BenchmarkTaskCreation(b *testing.B) {
	// Simulated task manager
	tasks := make(map[string]interface{})
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			taskID := fmt.Sprintf("task-%d", rand.Int63())
			task := map[string]interface{}{
				"id":          taskID,
				"title":       "Benchmark Task",
				"description": "Task created for benchmarking",
				"priority":    "normal",
				"status":      "pending",
				"created_at":  time.Now(),
			}

			mu.Lock()
			tasks[taskID] = task
			mu.Unlock()
		}
	})

	b.ReportMetric(float64(len(tasks))/b.Elapsed().Seconds(), "tasks/sec")
}

// BenchmarkTaskRetrieval measures task retrieval performance
func BenchmarkTaskRetrieval(b *testing.B) {
	// Setup: Create 10000 tasks
	tasks := make(map[string]interface{})
	taskIDs := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		tasks[taskID] = map[string]interface{}{
			"id":     taskID,
			"title":  fmt.Sprintf("Task %d", i),
			"status": "pending",
		}
		taskIDs[i] = taskID
	}

	var mu sync.RWMutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			taskID := taskIDs[rand.Intn(len(taskIDs))]
			mu.RLock()
			_ = tasks[taskID]
			mu.RUnlock()
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "retrievals/sec")
}

// BenchmarkJSONMarshaling measures JSON encoding performance
func BenchmarkJSONMarshaling(b *testing.B) {
	task := map[string]interface{}{
		"id":          "task-123",
		"title":       "Benchmark Task",
		"description": "Large description for benchmarking JSON marshaling performance",
		"priority":    "high",
		"status":      "running",
		"metadata": map[string]interface{}{
			"tags":      []string{"benchmark", "test", "performance"},
			"estimates": map[string]int{"cpu": 100, "memory": 512, "time": 300},
		},
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(task)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "marshals/sec")
}

// BenchmarkJSONUnmarshaling measures JSON decoding performance
func BenchmarkJSONUnmarshaling(b *testing.B) {
	task := map[string]interface{}{
		"id":          "task-123",
		"title":       "Benchmark Task",
		"description": "Large description",
		"priority":    "high",
		"status":      "running",
		"metadata": map[string]interface{}{
			"tags": []string{"benchmark", "test", "performance"},
		},
	}

	jsonData, _ := json.Marshal(task)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded map[string]interface{}
		err := json.Unmarshal(jsonData, &decoded)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "unmarshals/sec")
}

// BenchmarkConcurrentMapAccess measures concurrent map access performance
func BenchmarkConcurrentMapAccess(b *testing.B) {
	data := make(map[string]string)
	for i := 0; i < 1000; i++ {
		data[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	var mu sync.RWMutex
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := keys[rand.Intn(len(keys))]
			mu.RLock()
			_ = data[key]
			mu.RUnlock()
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "accesses/sec")
}

// BenchmarkChannelThroughput measures channel throughput
func BenchmarkChannelThroughput(b *testing.B) {
	ch := make(chan int, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Producer
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ch <- 1:
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		<-ch
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "messages/sec")
}

// BenchmarkGoroutineCreation measures goroutine creation overhead
func BenchmarkGoroutineCreation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			done := make(chan struct{})
			go func() {
				close(done)
			}()
			<-done
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "goroutines/sec")
}

// BenchmarkStringConcatenation measures string concatenation performance
func BenchmarkStringConcatenation(b *testing.B) {
	parts := []string{"Hello", " ", "World", " ", "from", " ", "HelixCode"}

	b.Run("Operator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := ""
			for _, part := range parts {
				result += part
			}
			_ = result
		}
	})

	b.Run("Builder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var builder bytes.Buffer
			for _, part := range parts {
				builder.WriteString(part)
			}
			_ = builder.String()
		}
	})

	b.Run("Sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := fmt.Sprintf("%s%s%s%s%s%s%s", parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], parts[6])
			_ = result
		}
	})
}

// BenchmarkMemoryAllocation measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 64)
		}
		b.ReportAllocs()
	})

	b.Run("Medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 4096)
		}
		b.ReportAllocs()
	})

	b.Run("Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 1024*1024)
		}
		b.ReportAllocs()
	})
}

// BenchmarkContextCancellation measures context cancellation overhead
func BenchmarkContextCancellation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			<-ctx.Done()
		}
	})
}

// BenchmarkTaskPriorityQueue simulates priority queue operations
func BenchmarkTaskPriorityQueue(b *testing.B) {
	type Task struct {
		ID       string
		Priority int
	}

	tasks := make([]Task, 0, 1000)
	for i := 0; i < 1000; i++ {
		tasks = append(tasks, Task{
			ID:       fmt.Sprintf("task-%d", i),
			Priority: rand.Intn(5),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simple insertion sort (would use heap in production)
		sorted := make([]Task, len(tasks))
		copy(sorted, tasks)

		for i := 1; i < len(sorted); i++ {
			key := sorted[i]
			j := i - 1
			for j >= 0 && sorted[j].Priority < key.Priority {
				sorted[j+1] = sorted[j]
				j--
			}
			sorted[j+1] = key
		}
	}
}

// BenchmarkConcurrentWorkers simulates worker pool performance
func BenchmarkConcurrentWorkers(b *testing.B) {
	workerCount := 10
	jobs := make(chan int, 100)
	results := make(chan int, 100)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Simulate work
				time.Sleep(time.Microsecond)
				results <- job * 2
			}
		}()
	}

	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	for i := 0; i < b.N; i++ {
		<-results
	}

	close(results)
	wg.Wait()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "jobs/sec")
}

// BenchmarkDatabaseMock simulates database operations
func BenchmarkDatabaseMock(b *testing.B) {
	// Mock database
	db := make(map[string]map[string]interface{})
	var mu sync.RWMutex

	b.Run("Insert", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				id := fmt.Sprintf("record-%d", rand.Int63())
				record := map[string]interface{}{
					"id":         id,
					"data":       "sample data",
					"created_at": time.Now(),
				}

				mu.Lock()
				db[id] = record
				mu.Unlock()
			}
		})
	})

	b.Run("Select", func(b *testing.B) {
		// Pre-populate
		for i := 0; i < 1000; i++ {
			id := fmt.Sprintf("record-%d", i)
			db[id] = map[string]interface{}{
				"id":   id,
				"data": "sample data",
			}
		}

		keys := make([]string, 0, len(db))
		for k := range db {
			keys = append(keys, k)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := keys[rand.Intn(len(keys))]
				mu.RLock()
				_ = db[key]
				mu.RUnlock()
			}
		})
	})

	b.Run("Update", func(b *testing.B) {
		keys := make([]string, 0, len(db))
		for k := range db {
			keys = append(keys, k)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := keys[rand.Intn(len(keys))]
				mu.Lock()
				if record, ok := db[key]; ok {
					record["updated_at"] = time.Now()
					db[key] = record
				}
				mu.Unlock()
			}
		})
	})
}
