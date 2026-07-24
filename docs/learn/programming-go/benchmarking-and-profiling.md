---
slug: benchmarking-and-profiling
title: Benchmarking & profiling
description: "Measuring performance with Go benchmarks and pprof — finding the hot path before optimizing it, the discipline real-time DSP code demands."
keywords: go benchmark, testing.B, b.N, go test -bench, pprof, go profiling, cpu profile, benchstat, performance go
level: advanced
status: full
prereq:
  - testing-in-go
faq:
  - q: What is b.N in a Go benchmark?
    a: "It's the number of iterations the testing framework chooses. Your benchmark loops for i := 0; i < b.N; i++ and the runner keeps raising b.N until the timing is statistically stable, then reports nanoseconds per operation. You never set b.N yourself — you just loop over it."
  - q: Do I need an external profiler for Go?
    a: "No. The runtime and the pprof package ship with the toolchain. go test -cpuprofile writes a profile, and go tool pprof reads it — showing which functions burned the most time, right down to a flame graph in your browser. It's all built in."
---

# Benchmarking & profiling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A benchmark is a **`func BenchmarkX(b *testing.B)`** that loops **`b.N`** times; run it
with **`go test -bench=.`** to get nanoseconds and allocations per operation. To find
*where* time goes, capture a **pprof** profile and read it with **`go tool pprof`**.
The rule: **measure before you optimize** — profile to find the hot path, then improve
only that.
</div>

Guessing which code is slow is usually wrong. This lesson covers Go's built-in tools
for measuring, which sit right alongside the
[testing package](/learn/programming-go/testing-in-go/) you already know.

## Writing a benchmark

A benchmark lives in a `_test.go` file and takes `*testing.B`. You loop `b.N` times;
the framework picks `b.N` to get a stable measurement:

```go
func BenchmarkDemodulate(b *testing.B) {
    samples := make([]complex64, 4096)
    d := NewC4FMDemod()
    b.ResetTimer()               // ignore setup time above
    for i := 0; i < b.N; i++ {
        d.Demodulate(samples)
    }
}
```

Run it and read the report:

```
$ go test -bench=Demodulate -benchmem
BenchmarkDemodulate-8   32418   36907 ns/op   1024 B/op   2 allocs/op
```

That's iterations run, time per call, bytes allocated per call, and allocation count
— the four numbers that matter for a DSP hot path.

## Profiling with pprof

A benchmark tells you *how slow*; a profile tells you *where*. Capture a CPU profile
and open it:

```
$ go test -bench=Pipeline -cpuprofile cpu.out
$ go tool pprof cpu.out
(pprof) top          # functions by time spent
(pprof) web          # flame graph in the browser
```

The `top` list and flame graph point straight at the function eating the cycles —
often not the one you'd have guessed. There are memory (`-memprofile`) and block
profiles too, read the same way.

| Flag / tool | What it measures |
|-------------|-----------------|
| `-bench=.` | Time per operation |
| `-benchmem` | Bytes and allocations per operation |
| `-cpuprofile cpu.out` | Where CPU time is spent |
| `-memprofile mem.out` | Where memory is allocated |
| `go tool pprof` | Reads any profile — `top`, `list`, `web` |

## Measure, then optimize

The discipline is simple and it's the whole point:

1. **Benchmark** the code so you have a baseline number.
2. **Profile** to find the actual hot path.
3. Change *that* — and only that.
4. Re-run the benchmark to confirm the number improved (`benchstat` compares two runs
   for you and flags noise).

Optimizing without step 2 usually speeds up code that was never the problem. For a
real-time SDR engine that must keep pace with incoming samples, this loop is how you
know a change helped rather than hoped it did.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the framework sets b.N; you loop over it." markdown="0">
  <p class="knowledge-check__q">Quick check: who decides the value of b.N in a benchmark?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You set it to a fixed count</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The testing framework raises it until timing is stable</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's always exactly one million</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A benchmark is **`func BenchmarkX(b *testing.B)`** looping **`b.N`** times, run with
  **`go test -bench`**.
- **`-benchmem`** adds bytes and allocations per operation.
- **pprof** (`-cpuprofile` + `go tool pprof`) shows *where* time goes.
- Always **measure, then optimize** — profile the hot path before changing it.

Next up: finding bugs with the debugger, the race detector, and static analysis.
