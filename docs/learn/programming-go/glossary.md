---
slug: glossary
title: Glossary of Go terms
description: Plain-language definitions of Go programming terms — goroutine, channel, interface, slice, map, struct, method, package, module, and more — each cross-linked to the lesson where it's explained.
keywords: go glossary, golang terms, go terminology, goroutine channel interface definition, go dictionary, slice map struct method
level: beginner
status: full
lesson_standalone: true
---

# Glossary of Go terms

Every term used across the [Go module](/learn/programming-go/), defined in plain
language and linked to the lesson where it's explained in full. Skim it as a
refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are
grouped by theme.

## Language basics

**Go (Golang)** — A statically typed, compiled, garbage-collected programming
language built at Google for simple, fast systems software. See [Why Go?](/learn/programming-go/why-go/)

**Compiled** — Turned into native machine code ahead of time, rather than
interpreted as it runs. Go programs compile to a single binary. See [Why Go?](/learn/programming-go/why-go/)

**Statically typed** — Every value's type is known at compile time, so type errors
are caught before the program runs. See [Values, variables & types](/learn/programming-go/values-and-types/)

**Binary** — The single self-contained executable file `go build` produces, runnable
without Go installed. See [Your first Go program](/learn/programming-go/hello-go/)

**`package main`** — The special package that produces a runnable command; it must
contain a `func main()` entry point. See [Your first Go program](/learn/programming-go/hello-go/)

**Zero value** — The default value a variable takes when declared without one: `0`,
`false`, `""`, or `nil`. See [Values, variables & types](/learn/programming-go/values-and-types/)

**`const`** — A declaration for a value that never changes. See [Values, variables & types](/learn/programming-go/values-and-types/)

## The toolchain

**`go build` / `go run`** — Compile to a keepable binary, or compile-and-run once.
See [The Go toolchain](/learn/programming-go/the-go-toolchain/)

**`gofmt`** — The tool that rewrites code into Go's one official style. See [The Go toolchain](/learn/programming-go/the-go-toolchain/)

**`go vet`** — A tool that flags suspicious, likely-buggy code that still compiles.
See [The Go toolchain](/learn/programming-go/the-go-toolchain/)

**`go.mod`** — The file that defines a module and pins its dependency versions. See
[Packages & modules](/learn/programming-go/packages-and-modules/)

## Functions and types

**Function** — A named block of code that takes parameters and can return values;
Go functions can return **multiple** values. See [Functions & error handling](/learn/programming-go/functions-and-errors/)

**`error`** — The built-in interface a fallible function returns; `nil` means
success. See [Functions & error handling](/learn/programming-go/functions-and-errors/)

**`if err != nil`** — The standard idiom for handling a failed call. See [Functions & error handling](/learn/programming-go/functions-and-errors/)

**`defer`** — Schedules a call to run when the surrounding function returns, used for
cleanup. See [Functions & error handling](/learn/programming-go/functions-and-errors/)

**Struct** — A type that groups named fields together. See [Structs & methods](/learn/programming-go/structs-and-methods/)

**Method** — A function attached to a type through a **receiver**. See [Structs & methods](/learn/programming-go/structs-and-methods/)

**Receiver** — The type a method belongs to; a **pointer receiver** (`*T`) can modify
the value, a **value receiver** (`T`) works on a copy. See [Structs & methods](/learn/programming-go/structs-and-methods/)

**Embedding** — Placing one struct (or interface) inside another to reuse its fields
and methods — Go's composition in place of inheritance. See [Structs & methods](/learn/programming-go/structs-and-methods/)

**Interface** — A set of method signatures; a type satisfies it **implicitly** by
having those methods. See [Interfaces & composition](/learn/programming-go/interfaces/)

**`io.Reader`** — The one-method standard interface for "something you can read bytes
from," satisfied by files, sockets, buffers, and more. See [The standard library](/learn/programming-go/the-standard-library/)

## Concurrency

**Concurrency** — Structuring a program as independently running tasks (distinct from
parallelism, running them literally at once). See [Goroutines](/learn/programming-go/goroutines/)

**Goroutine** — A function launched with the `go` keyword that runs concurrently;
lightweight and scheduled by Go's runtime. See [Goroutines](/learn/programming-go/goroutines/)

**Channel** — A typed pipe that passes values between goroutines; `ch <- v` sends,
`<-ch` receives. See [Channels](/learn/programming-go/channels/)

**Buffered / unbuffered channel** — Unbuffered channels make sender and receiver
rendezvous; buffered channels hold a set number of values. See [Channels](/learn/programming-go/channels/)

**Data race** — A bug where goroutines access the same variable at once with no
synchronization; caught by `go test -race`. See [Channels](/learn/programming-go/channels/)

**`select`** — A statement that waits on multiple channel operations and proceeds with
whichever is ready first. See [select & synchronization](/learn/programming-go/select-and-sync/)

**`sync.Mutex`** — A lock that guards shared state so only one goroutine enters a
critical section at a time. See [select & synchronization](/learn/programming-go/select-and-sync/)

**`sync.WaitGroup`** — A counter that lets one goroutine wait for a group of others to
finish. See [select & synchronization](/learn/programming-go/select-and-sync/)

**`context`** — A value that carries cancellation and deadlines across a call tree.
See [select & synchronization](/learn/programming-go/select-and-sync/)

## Code organization

**Package** — A directory of Go files compiled together and imported as a unit. See
[Packages & modules](/learn/programming-go/packages-and-modules/)

**Module** — A versioned collection of packages defined by `go.mod`. See [Packages & modules](/learn/programming-go/packages-and-modules/)

**Exported / unexported** — A capitalized name is exported (visible to other
packages); a lowercase name is private to its package. See [Packages & modules](/learn/programming-go/packages-and-modules/)

**`internal/`** — A directory whose packages can only be imported within the same
module. See [Packages & modules](/learn/programming-go/packages-and-modules/)

## Collections and generics

**Slice** — A growable, ordered view into an array, extended with `append`. See [Slices, maps & generics](/learn/programming-go/slices-maps-generics/)

**Map** — A key/value lookup table; the `value, ok` form reports whether a key is
present. See [Slices, maps & generics](/learn/programming-go/slices-maps-generics/)

**`append` / `make` / `copy`** — Built-ins to grow a slice, allocate a slice/map, and
copy slice contents. See [Slices, maps & generics](/learn/programming-go/slices-maps-generics/)

**Generics (type parameters)** — Writing one function or type that works across many
types safely, added in Go 1.18. See [Slices, maps & generics](/learn/programming-go/slices-maps-generics/)

## Testing

**`go test`** — The command that runs tests in `_test.go` files. See [Testing in Go](/learn/programming-go/testing-in-go/)

**Table-driven test** — A test that lists its cases as data and loops over them,
Go's dominant testing style. See [Testing in Go](/learn/programming-go/testing-in-go/)

**Benchmark** — A `func BenchmarkX(b *testing.B)` that measures performance with `go
test -bench`. See [Testing in Go](/learn/programming-go/testing-in-go/)

**Standard library** — The large set of packages Go ships with — `io`, `net/http`,
`encoding/json`, and many more — reducing the need for dependencies. See [The standard library](/learn/programming-go/the-standard-library/)

## Control flow & memory

**Control flow** — Go's small set of control structures: `if`, a single `for` loop
that covers every looping case, and `switch`. See [Control flow](/learn/programming-go/control-flow/)

**Pointer** — A value holding the address of another value; Go has pointers (`*T`, `&x`)
but no pointer arithmetic. See [Pointers](/learn/programming-go/pointers/)

## Data & text

**Rune** — Go's name for a Unicode code point (an `int32`); ranging over a string yields
runes, not bytes. See [Strings, bytes & runes](/learn/programming-go/strings-and-runes/)

**Struct tag** — A string annotation on a struct field (like `json:"name"`) that guides
encoding such as JSON. See [JSON & serialization](/learn/programming-go/json-and-serialization/)

**Marshal / Unmarshal** — Converting a Go value to JSON (marshal) and back (unmarshal)
with `encoding/json`. See [JSON & serialization](/learn/programming-go/json-and-serialization/)

**bufio** — Buffered wrappers around readers and writers for efficient I/O. See [Working with files & I/O](/learn/programming-go/working-with-files/)

## Concurrency

**Context** — A value carrying cancellation and deadlines through a call tree; passed as
the first argument by convention. See [Context & cancellation](/learn/programming-go/context-and-cancellation/)

**Worker pool** — A fixed set of goroutines pulling jobs off a channel — a core
concurrency pattern. See [Concurrency patterns](/learn/programming-go/concurrency-patterns/)

**Pipeline / fan-out / fan-in** — Composing stages connected by channels, splitting work
across goroutines and merging results. See [Concurrency patterns](/learn/programming-go/concurrency-patterns/)

## Dependencies, errors & quality

**`go.sum`** — The file recording cryptographic checksums of dependencies for
reproducible, verifiable builds. See [Dependency management](/learn/programming-go/dependency-management/)

**Error wrapping** — Adding context to an error with `fmt.Errorf("...: %w", err)`,
inspected later with `errors.Is` / `errors.As`. See [Error-handling patterns](/learn/programming-go/error-handling-patterns/)

**Sentinel error** — A predefined error value (like `io.EOF`) compared against with
`errors.Is`. See [Error-handling patterns](/learn/programming-go/error-handling-patterns/)

**Benchmark / pprof** — A `func BenchmarkX(b *testing.B)` measuring performance, and the
profiler for finding hot paths. See [Benchmarking & profiling](/learn/programming-go/benchmarking-and-profiling/)

**delve** — The Go debugger (`dlv`) for stepping through code and inspecting state. See
[Debugging Go](/learn/programming-go/debugging-go/)

**`log/slog`** — The standard structured-logging package: leveled log records with
key/value attributes. See [Logging with slog](/learn/programming-go/logging-and-slog/)

## Idioms & structure

**`cmd/` and `internal/`** — Conventional directories for entry points and private
packages in a Go repository. See [Structuring a Go project](/learn/programming-go/project-structure/)

**Idiomatic Go** — Code written the community way: gofmt-formatted, explicit errors,
small interfaces, clear names. See [Writing idiomatic Go](/learn/programming-go/idiomatic-go/)
