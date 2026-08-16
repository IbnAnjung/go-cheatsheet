package workerpool

import (
	"runtime"
	"testing"
	"time"
)

func TestProcessJobs(t *testing.T) {
	numJobs := 100
	numWorkers := 5

	var jobs []Job
	for i := 1; i <= numJobs; i++ {
		jobs = append(jobs, Job{ID: i, Value: i})
	}

	startGoroutines := runtime.NumGoroutine()

	results := ProcessJobs(jobs, numWorkers)

	if len(results) != numJobs {
		t.Fatalf("Expected %d results, got %d", numJobs, len(results))
	}

	// Verify all results are correct
	resultMap := make(map[int]int)
	for _, r := range results {
		resultMap[r.JobID] = r.Square
	}

	for _, j := range jobs {
		expected := j.Value * j.Value
		if resultMap[j.ID] != expected {
			t.Errorf("For Job ID %d, expected square %d, got %d", j.ID, expected, resultMap[j.ID])
		}
	}

	// Cek Goroutine leak
	time.Sleep(100 * time.Millisecond) // Beri waktu sedikit untuk cleanup goroutine
	endGoroutines := runtime.NumGoroutine()

	// Toleransi kecil jika ada goroutine internal testing Go yang berjalan
	if endGoroutines > startGoroutines+2 {
		t.Errorf("Potential Goroutine leak! Start: %d, End: %d", startGoroutines, endGoroutines)
	}
}
