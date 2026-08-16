package workerpool

import "sync"

// Job represents a single unit of work.
type Job struct {
	ID    int
	Value int
}

// Result represents the outcome of a processed Job.
type Result struct {
	JobID  int
	Square int
}

// ProcessJobs memproses seluruh 'jobs' secara konkuren menggunakan tepat sejumlah 'numWorkers' goroutine.
// Fungsi ini harus mengembalikan slice berisi Result, di mana 'Square' adalah hasil kuadrat dari 'Value' pada Job.
// Urutan Result di dalam slice tidak harus berurutan.
// PASTIKAN: Tidak ada goroutine yang leak (berjalan terus) setelah fungsi mengembalikan hasil!
func ProcessJobs(jobs []Job, numWorkers int) []Result {
	jobChans := make(chan Job, len(jobs))
	resultChans := make(chan Result)

	result := make([]Result, 0, len(jobs))

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChans {
				square := job.Value * job.Value

				resultChans <- Result{
					JobID:  job.ID,
					Square: square,
				}
			}
		}()
	}

	for _, job := range jobs {
		jobChans <- job
	}

	close(jobChans)

	go func() {
		wg.Wait()
		close(resultChans)
	}()

	for res := range resultChans {
		result = append(result, res)
	}

	return result
}
