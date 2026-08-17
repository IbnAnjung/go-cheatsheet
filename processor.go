package aggregator

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// LogEntry adalah bentuk struktur dari JSON log mentah yang masuk.
// Di dunia nyata, ini bisa berisi puluhan field.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Message   string `json:"message"`
}

// ProcessLogs_Bad mensimulasikan layanan pemrosesan jutaan log dari Kafka/RabbitMQ.
// Menerima data mentah byte JSON, lalu mengubahnya menjadi format string siap tulis ke file.
// Format output: "[2026-08-17T10:00:00Z] ERROR (billing-svc): Transaction failed"
func ProcessLogs_Bad(rawLogs [][]byte) []string {
	var results []string

	for _, raw := range rawLogs {
		// ALOKASI BARU 1: Membuat struct pointer setiap kali perulangan.
		// Karena kita lempar ini ke json.Unmarshal (yang parameter argumennya `any`),
		// compiler terpaksa menaruhnya di Heap (escape analysis memory leak).
		entry := &LogEntry{}

		err := json.Unmarshal(raw, entry)
		if err != nil {
			continue // skip error log
		}

		// ALOKASI BARU 2 & 3: String concatenation pakai + atau fmt.Sprintf
		// Ini memicu rentetan pembuatan string baru yang membuang-buang memori.
		formatted := fmt.Sprintf("[%s] %s (%s): %s",
			entry.Timestamp, entry.Level, entry.Service, entry.Message)

		results = append(results, formatted)
	}

	return results
}

// TODO: Buat fungsi ProcessLogs_Good di bawah ini
// Clue:
// - Bikin pool untuk objek *LogEntry
// - Bikin pool untuk objek *strings.Builder
var logEntryPool = &sync.Pool{
	New: func() any {
		return &LogEntry{}
	},
}

var sbPool = &sync.Pool{
	New: func() any {
		return new(strings.Builder)
	},
}

func ProcessLogs_Good(rawLogs [][]byte) []string {
	results := make([]string, len(rawLogs))
	totalWorker := 3
	eachWorker := (len(rawLogs) + totalWorker - 1) / totalWorker

	var wg sync.WaitGroup
	for i := 0; i < totalWorker; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := i * eachWorker
			end := start + eachWorker

			if start >= len(rawLogs) {
				return
			}

			if end >= len(rawLogs) {
				end = len(rawLogs)
			}

			for j, log := range rawLogs[start:end] {
				lEntry := logEntryPool.Get().(*LogEntry)
				if err := json.Unmarshal(log, lEntry); err != nil {
					continue
				}

				sb := sbPool.Get().(*strings.Builder)
				sb.WriteByte('[')
				sb.WriteString(lEntry.Timestamp)
				sb.WriteByte(']')

				sb.WriteByte(' ')
				sb.WriteString(lEntry.Level)
				sb.WriteByte(' ')

				sb.WriteByte('(')
				sb.WriteString(lEntry.Service)
				sb.WriteString("): ")

				sb.WriteString(lEntry.Message)

				results[start+j] = sb.String()

				*lEntry = LogEntry{}
				logEntryPool.Put(lEntry)
				sb.Reset()
				sbPool.Put(sb)
			}
		}(i)
	}

	wg.Wait()

	// for _, log := range rawLogs {
	// 	lEntry := logEntryPool.Get().(*LogEntry)
	// 	if err := json.Unmarshal(log, lEntry); err != nil {
	// 		continue
	// 	}

	// 	sb := sbPool.Get().(*strings.Builder)
	// 	sb.WriteByte('[')
	// 	sb.WriteString(lEntry.Timestamp)
	// 	sb.WriteByte(']')

	// 	sb.WriteByte(' ')
	// 	sb.WriteString(lEntry.Level)
	// 	sb.WriteByte(' ')

	// 	sb.WriteByte('(')
	// 	sb.WriteString(lEntry.Service)
	// 	sb.WriteString("): ")

	// 	sb.WriteString(lEntry.Message)

	// 	results = append(results, sb.String())

	// 	*lEntry = LogEntry{}
	// 	logEntryPool.Put(lEntry)
	// 	sb.Reset()
	// 	sbPool.Put(sb)
	// }

	return results
}
