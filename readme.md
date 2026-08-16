# Advanced Go Cheatsheet

A practical collection of **Advanced Go (Golang) patterns, memory management, garbage collection (GC) tuning, and concurrency strategies**.

This repository is designed for intermediate to advanced developers looking to deepen their understanding of topics that are often overlooked, such as escape analysis, zero-allocation techniques, and complex concurrency patterns.

## Contents

### Concurrency Patterns & Synchronization

### Memory Management & Garbage Collection

### Performance Optimization

### Often Overlooked Topics

---

## Repository Structure

```text
go-cheatsheet/
│
├── concurrency/
│   ├── worker-pool/
│   ├── fan-in-out/
│   ├── pipeline/
│   └── errgroup/
├── memory/
│   ├── escape-analysis/
│   ├── sync-pool/
│   ├── gc-tuning/
│   └── alignment/
├── performance/
│   ├── pprof/
│   ├── tracing/
│   └── zero-alloc/
└── ignored-topics/
    └── graceful-shutdown/
```

Each topic contains concise explanations, deep dives into the "why", and runnable examples.

---

## Philosophy

The goal of this repository is **deep, practical understanding**.

Each topic should preferably contain:
1. A clear explanation of the underlying mechanics.
2. A minimal, focused example.
3. Common pitfalls and anti-patterns.
4. Profiling or benchmarking proof (e.g., comparing allocations).
5. A recommended best practice.

Example: `sync.Pool` should not only explain how to `Get()` and `Put()`, but also demonstrate the reduction in GC pressure via benchmarks and explain when *not* to use it (e.g., short-lived, small objects).

---

## Running Examples

Most examples should be runnable using the standard Go toolchain.

```bash
go run .
```

Run tests with race detection:
```bash
go test -race ./...
```

Run benchmarks (crucial for memory/performance topics):
```bash
go test -bench=. -benchmem ./...
```

---

## Contributing

Contributions are welcome, especially if they involve deep dives into Go internals or real-world performance tuning case studies.

When adding a new cheatsheet, keep it:
* Accurate and backed by data (benchmarks/profiling).
* Concise but thorough on the "why".
* Runnable.
* Focused on satu topik lanjutan (advanced topic).
