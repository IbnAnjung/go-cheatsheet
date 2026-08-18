package parser

import (
	"errors"
	"strings"
	"sync"
)

var ErrInvalidFormat = errors.New("invalid log format")

type LogEntry struct {
	Level   string
	Message string
}

var logEntryPool = sync.Pool{
	New: func() any {
		return new(LogEntry)
	},
}

// ParseLogLine menerima baris log (contoh: "[INFO] Something happened")
// dan memecahnya menjadi LogEntry.
// TODO: Optimasi fungsi ini menggunakan sync.Pool agar terhindar dari alokasi &LogEntry{} baru terus-menerus!
func ParseLogLine(line string) (*LogEntry, error) {
	if !strings.HasPrefix(line, "[") {
		return nil, ErrInvalidFormat
	}

	endIdx := strings.Index(line, "]")
	if endIdx == -1 {
		return nil, ErrInvalidFormat
	}

	level := line[1:endIdx]

	// Spasi setelah ']'
	msgStart := endIdx + 1
	if msgStart < len(line) && line[msgStart] == ' ' {
		msgStart++
	}

	message := line[msgStart:]

	// ALLOCATION: Karena me-return pointer, struct ini escape ke heap!
	// Gunakan sync.Pool untuk menghindari alokasi baru setiap dipanggil.
	entry := logEntryPool.Get().(*LogEntry)
	entry.Level = level
	entry.Message = message

	return entry, nil
}

// TODO: Buatlah fungsi ReleaseLogEntry(entry *LogEntry)
// untuk mengembalikan entry kembali ke dalam sync.Pool agar bisa didaur ulang.
func ReleaseLogEntry(entry *LogEntry) {
	entry.Level = ""
	entry.Message = ""
	logEntryPool.Put(entry)
}
