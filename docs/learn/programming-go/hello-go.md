---
slug: hello-go
title: Your first Go program
description: Install Go, write a hello-world program, and understand the difference between go run and go build — how Go turns source code into a single static binary.
keywords: go hello world, install go, go run, go build, first go program, package main, func main, go binary
level: beginner
status: full
prereq:
  - why-go
faq:
  - q: What is the difference between go run and go build?
    a: "`go run` compiles your program to a temporary location and immediately runs it, which is handy while developing. `go build` compiles it to a permanent binary file in your current directory that you can run again later or copy to another machine. Both do the same compilation; they differ only in whether the result is thrown away or kept."
  - q: Why does every Go program start with package main?
    a: A Go program that produces a runnable command must live in a package named `main` and contain a function named `main`. That function is the entry point the compiler wires up as the start of the program. Packages named anything else are libraries — reusable code meant to be imported, not run directly.
---

# Your first Go program

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A runnable Go program needs a **`package main`** declaration and a **`func main()`**
entry point. You compile-and-run it with **`go run`** while developing, or compile
it to a permanent binary with **`go build`**. That binary is self-contained — copy
it to another machine and it just runs.
</div>

Enough theory — let's make Go do something. This lesson gets Go installed, gets a
program running, and explains the handful of lines every Go program starts with.

## Installing Go

Download the installer for your operating system from the official site
(go.dev/dl) and follow the prompts, or use your package manager
(`brew install go`, `apt install golang`, and so on — see the
[Linux CLI module](/learn/linux-cli/package-management/) if package managers are
new to you). Confirm it worked by opening a terminal and running:

```text
$ go version
go version go1.22.0 linux/amd64
```

If you see a version number, you are ready.

## The smallest useful program

Create a file called `hello.go` and put this in it:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, radio!")
}
```

Four ideas are packed into six lines:

- **`package main`** — this file belongs to the special package that produces a
  runnable command.
- **`import "fmt"`** — pull in the standard library's *format* package, which
  handles printing.
- **`func main()`** — the entry point; execution starts here.
- **`fmt.Println(...)`** — call the `Println` function from `fmt` to print a line.

## Running it

From the same folder, run:

```text
$ go run hello.go
Hello, radio!
```

`go run` compiled your code and ran it in one step. Nothing was left behind — great
for quick iteration.

## Building a binary

When you want a program you can keep and share, build it instead:

```text
$ go build hello.go
$ ./hello
Hello, radio!
```

Now there is a `hello` file (or `hello.exe` on Windows) sitting in your folder.
That single file *is* your program — it has no dependency on Go being installed on
the machine that runs it. This is the "single static binary" promise from lesson 1,
and it is why distributing a Go tool is often just "here is the file."

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="Flow from hello.go source through the Go compiler to a single binary that runs on any machine." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="40" width="120" height="40" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="70" y="65" text-anchor="middle" font-size="13" fill="currentColor">hello.go</text>
  <line x1="130" y1="60" x2="200" y2="60" stroke="currentColor" stroke-width="1.5" marker-end="url(#ar)"/>
  <rect x="200" y="40" width="120" height="40" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="260" y="65" text-anchor="middle" font-size="13" fill="currentColor">go build</text>
  <line x1="320" y1="60" x2="390" y2="60" stroke="currentColor" stroke-width="1.5" marker-end="url(#ar)"/>
  <rect x="390" y="40" width="120" height="40" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="450" y="65" text-anchor="middle" font-size="13" fill="currentColor">./hello</text>
  <defs><marker id="ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Source in, one binary out — the whole build pipeline for a simple Go program.</figcaption>
</figure>

## A note on formatting

Save your file and run `gofmt -w hello.go`. It rewrites the file into Go's one
official style — indentation, spacing, and all. Go settles formatting debates by
having exactly one answer, and every editor can run `gofmt` for you on save. You
will meet the rest of the toolchain in the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — go run compiles and runs without leaving a binary behind." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to test a change quickly without keeping the compiled output. Which do you use?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">go run</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">go build</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">gofmt</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A runnable program needs **`package main`** and **`func main()`**.
- **`import`** pulls in packages like `fmt` from the standard library.
- **`go run`** compiles and runs in one step; **`go build`** produces a keepable,
  self-contained **binary**.
- **`gofmt`** enforces Go's single official code style.

Next up: the rest of the toolchain that formats, checks, tests, and builds your code.
