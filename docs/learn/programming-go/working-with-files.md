---
slug: working-with-files
title: Working with files & I/O
description: "Reading and writing files and streams with os, io, and bufio — and the reader/writer interfaces that make every source and sink in Go composable."
keywords: go file io, os.Open, os.Create, io.Reader, io.Writer, bufio, defer close, go read file, go write file
level: intermediate
status: full
prereq:
  - interfaces
faq:
  - q: Why do I always see defer file.Close() in Go?
    a: "Because an open file holds an OS resource that must be released. defer schedules Close to run when the function returns — whether it returns normally or early on an error — so the file is always closed and you never leak a handle. It's the standard Go cleanup pattern."
  - q: What's the point of bufio?
    a: "Raw file reads and writes each hit the operating system, which is slow when done a byte or line at a time. bufio wraps a reader or writer with an in-memory buffer, so many small operations become a few big ones. bufio.Scanner also makes line-by-line reading trivial."
---

# Working with files & I/O

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Open files with **`os.Open`** (read) and **`os.Create`** (write), and always
**`defer f.Close()`**. The real power is the **`io.Reader`** and **`io.Writer`**
interfaces — one-method contracts that files, network connections, and buffers all
satisfy, so the same code works on any of them. **`bufio`** adds buffering and
convenient line scanning.
</div>

Reading and writing data is where the [interfaces](/learn/programming-go/interfaces/)
lesson pays off: Go models every stream through two tiny interfaces. This lesson
shows the everyday patterns.

## Opening and closing a file

`os.Open` returns a `*os.File` (which is an `io.Reader`) and an error. Pair every open
with a deferred close:

```go
f, err := os.Open("capture.cfile")
if err != nil {
    return fmt.Errorf("open capture: %w", err)
}
defer f.Close()   // runs when the function returns, even on error
```

`defer` guarantees cleanup runs no matter which path the function takes — the reason
you see it on the line right after every successful open.

## The two interfaces that tie it together

Almost all of Go's I/O is built on two one-method interfaces:

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
```

Because a file, a network socket, a byte buffer, and a gzip stream all satisfy these,
a function written against `io.Reader` works with every one of them. `io.Copy` is the
classic example — it pipes any reader into any writer:

```go
n, err := io.Copy(dst, src)   // dst is any Writer, src is any Reader
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A source satisfying io.Reader flows through io.Copy into a destination satisfying io.Writer." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="45" width="130" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="75" y="66" text-anchor="middle" font-size="11" fill="currentColor">src (io.Reader)</text>
  <rect x="195" y="45" width="120" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="255" y="66" text-anchor="middle" font-size="11" fill="currentColor">io.Copy</text>
  <rect x="380" y="45" width="130" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="445" y="66" text-anchor="middle" font-size="11" fill="currentColor">dst (io.Writer)</text>
  <line x1="140" y1="62" x2="195" y2="62" stroke="currentColor" stroke-width="1.5" marker-end="url(#ia)"/>
  <line x1="315" y1="62" x2="380" y2="62" stroke="currentColor" stroke-width="1.5" marker-end="url(#ia)"/>
  <defs><marker id="ia" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Code written against io.Reader/io.Writer works with any source and any sink — files, sockets, or in-memory buffers.</figcaption>
</figure>

## Buffering and scanning lines

Reading a file line by line is a `bufio.Scanner`:

```go
f, _ := os.Open("channels.txt")
defer f.Close()

sc := bufio.NewScanner(f)
for sc.Scan() {
    line := sc.Text()   // one line, no trailing newline
    addChannel(line)
}
if err := sc.Err(); err != nil {
    return err
}
```

For writing, wrap the file in a `bufio.Writer` so many small writes batch into a few
system calls — then `Flush` before the file closes:

```go
w := bufio.NewWriter(f)
defer w.Flush()
fmt.Fprintf(w, "%s,%.0f\n", ch.Name, ch.FreqHz)
```

## Whole-file shortcuts

When a file is small enough to hold in memory, `os.ReadFile` and `os.WriteFile`
collapse open/read/close into one call:

```go
data, err := os.ReadFile("scanlist.json")   // []byte of the whole file
err = os.WriteFile("out.json", data, 0o644)
```

This is how GopherTrunk loads a JSON scan list at startup — read the bytes, then
`json.Unmarshal` them into the config struct from the
[previous lesson](/learn/programming-go/json-and-serialization/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — defer Close releases the file whichever way the function returns." markdown="0">
  <p class="knowledge-check__q">Quick check: why put <em>defer f.Close()</em> right after opening a file?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It makes reads faster</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It guarantees the file is closed on every return path, including errors</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It reopens the file if a read fails</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Open with **`os.Open`**/**`os.Create`** and **`defer f.Close()`** immediately.
- **`io.Reader`** and **`io.Writer`** are one-method interfaces every stream
  satisfies, so I/O code composes.
- **`bufio`** buffers reads/writes and scans lines; remember to `Flush` a buffered
  writer.
- **`os.ReadFile`**/**`os.WriteFile`** handle a whole small file in one call.

Next up: Go's headline feature — launching concurrent work with goroutines.
