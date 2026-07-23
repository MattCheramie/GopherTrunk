---
slug: strings-and-runes
title: Strings, bytes & runes
description: "How Go handles text — immutable strings, byte slices, and runes — and why a character isn't always one byte once Unicode and UTF-8 enter the picture."
keywords: go strings, go runes, go bytes, utf-8 go, immutable string, rune int32, range over string, string to byte slice
level: intermediate
status: full
prereq:
  - slices-maps-generics
faq:
  - q: What is a rune in Go?
    a: "A rune is an alias for int32 and holds a single Unicode code point — one character, which may take several bytes in UTF-8. Ranging over a string yields runes, so a loop sees whole characters even when they occupy more than one byte."
  - q: Why can't I change a character in a Go string?
    a: "Strings are immutable — their bytes never change after creation, which makes them safe to share without copying. To edit text, convert to a []byte (or []rune), modify that, and convert back to a string."
---

# Strings, bytes & runes

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A Go **string** is an **immutable** sequence of bytes, usually holding **UTF-8**
text. Indexing a string gives you a **byte**, not a character. A **rune** (`int32`)
is one Unicode code point, and **`range`** over a string walks it rune by rune. To
edit text, convert to **`[]byte`** or **`[]rune`**, change it, and convert back.
</div>

Text looks simple until non-ASCII characters arrive. This lesson untangles the three
layers Go gives you — strings, bytes, and runes — building on
[Slices, maps & generics](/learn/programming-go/slices-maps-generics/).

## What is a Go string, really?

A string is a read-only slice of bytes with a length. It is immutable: once made, its
bytes never change, so passing a string around never copies the underlying data.

```go
s := "P25 phase 2"
fmt.Println(len(s)) // 11 — the number of bytes
fmt.Println(s[0])  // 80 — the byte value of 'P', not "P"
// s[0] = 'X'      // compile error: strings are immutable
```

Note that indexing yields a **byte** (`uint8`). For pure ASCII that happens to match
one character, but the moment text contains multi-byte characters, bytes and
characters part ways.

## Bytes vs runes

UTF-8 encodes each Unicode character as one to four bytes. ASCII stays one byte; an
accented letter or symbol takes more. A **rune** is Go's name for one code point:

```go
s := "µV"                    // micro-volts label
fmt.Println(len(s))          // 3 — µ is two bytes, V is one
fmt.Println(len([]rune(s)))  // 2 — two characters
```

| Type | What it is | Size | Use it for |
|------|-----------|------|-----------|
| `byte` | alias for `uint8` | 1 byte | Raw data, ASCII, I/O buffers |
| `rune` | alias for `int32` | 4 bytes | One Unicode character |
| `string` | immutable byte sequence | varies | Text you read and pass around |
| `[]byte` | mutable byte slice | varies | Text or binary you edit |

## Ranging over a string

A plain index walks bytes; `range` walks **runes**, decoding UTF-8 as it goes and
giving you each character's byte offset:

```go
for i, r := range "µVdBm" {
    fmt.Printf("%d: %c\n", i, r)  // i jumps by the byte width of each rune
}
```

This is the reliable way to iterate characters. Reach for the `unicode/utf8` and
`strings` packages from the [standard library](/learn/programming-go/the-standard-library/)
when you need counting, splitting, or case handling.

## Editing text

Because strings are immutable, you build new ones. For heavy concatenation use a
`strings.Builder` instead of `+=` so you don't allocate a fresh string each time:

```go
var b strings.Builder
for _, part := range []string{"460", ".", "025", " MHz"} {
    b.WriteString(part)
}
label := b.String()   // "460.025 MHz"
```

For in-place edits, convert to `[]byte` or `[]rune`, mutate, and convert back — each
conversion copies, which is exactly what preserves the original string's immutability.

<div class="knowledge-check" data-quiz data-correct-msg="Right — indexing a string gives a byte; range gives runes." markdown="0">
  <p class="knowledge-check__q">Quick check: what does <em>s[0]</em> return for a Go string s?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The first character as a string</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The first byte (a uint8)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The first rune (a code point)</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **string** is an **immutable** sequence of bytes, usually UTF-8 text.
- Indexing gives a **byte**; **`range`** gives **runes** (whole characters).
- A **`rune`** is `int32`, one Unicode code point; multi-byte characters make
  `len(string)` count bytes, not characters.
- Edit text via **`[]byte`**/**`[]rune`** or a `strings.Builder`.

Next up: turning Go values into JSON and back with the standard library.
