---
slug: logging-and-slog
title: Logging with slog
description: "Structured logging with the standard log/slog package — levels, key/value fields, and why a running service logs in structured form instead of plain printed lines."
keywords: go slog, log/slog, structured logging go, slog.Info, log levels go, structured logs, json logging go, slog attributes
level: intermediate
status: full
prereq:
  - the-standard-library
faq:
  - q: What makes slog "structured" logging?
    a: "Instead of baking values into a sentence, slog records them as key/value pairs: slog.Info(\"locked\", \"freq_hz\", 460025000, \"snr\", 19.7). Each field stays a separate, typed attribute, so a log system can filter, search, and aggregate on freq_hz or snr directly — something you can't reliably do with a formatted string."
  - q: Why not just use fmt.Println or the old log package?
    a: "For a quick script, Println is fine. But a long-running service needs log levels to control verbosity, machine-readable output for log aggregators, and consistent fields across every line. slog gives all three from the standard library, which is why it's the default choice for services."
---

# Logging with slog

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`log/slog`** is Go's standard **structured logging** package. Instead of formatting
values into a sentence, you attach them as **key/value attributes** —
`slog.Info("locked", "freq_hz", f, "snr", s)`. It supports **levels** (Debug, Info,
Warn, Error) and can emit human-readable text or **JSON** for log aggregators. A
running service logs this way so its output can be filtered and searched, not just
read.
</div>

A quick program can print. A service that runs for days needs logs it can query. This
lesson covers `log/slog`, part of the
[standard library](/learn/programming-go/the-standard-library/) since Go 1.21.

## Structured, not formatted

The old habit is to build a sentence:

```go
log.Printf("locked on %.4f MHz at %.1f dB SNR", freq/1e6, snr)  // one opaque string
```

`slog` records the same facts as separate, typed fields:

```go
slog.Info("signal locked",
    "freq_hz", freq,
    "snr", snr,
    "protocol", "P25",
)
```

Each key/value is preserved, so a log system can filter `protocol=P25` or graph `snr`
over time. That is the whole point of *structured* logging: the data stays data.

## Levels

`slog` has four built-in levels, and a logger can be set to drop anything below a
threshold — so you leave `Debug` lines in the code and simply raise the level in
production.

| Level | Use it for |
|-------|-----------|
| `slog.Debug` | Detailed tracing, off in production |
| `slog.Info` | Normal events — startup, a lock acquired |
| `slog.Warn` | Something unexpected but recoverable |
| `slog.Error` | A failure that needs attention |

```go
slog.Debug("fft frame", "bins", 4096)     // hidden unless level <= Debug
slog.Error("decode failed", "err", err)   // always shown
```

## Text or JSON output

`slog` separates *what* you log from *how* it's formatted. Choose a handler:

```go
// Human-readable text (development):
h := slog.NewTextHandler(os.Stderr, nil)

// Machine-readable JSON (production / aggregators):
h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})

slog.SetDefault(slog.New(h))
```

The JSON handler emits one object per line — exactly what a log collector wants. You
can also attach fields once with `logger.With("protocol", "P25")` so every subsequent
line carries them, ideal for a per-channel logger in a scanner. You'll see the same
structured output feed dashboards in
[Logging & health checks](/learn/deployment/logging-and-health-checks/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — slog keeps each value as a separate key/value attribute." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes slog output "structured"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It always writes to a database</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Values are logged as separate key/value attributes, not baked into a sentence</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It colourizes the terminal output</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`log/slog`** logs **key/value attributes**, keeping data queryable instead of
  formatted into a string.
- It has four **levels** — Debug, Info, Warn, Error — with a settable threshold.
- Swap **handlers** for text (dev) or **JSON** (production/aggregators).
- **`logger.With(...)`** attaches persistent fields to every line.

Next up: how a real Go repository is laid out — cmd, internal, and packages.
