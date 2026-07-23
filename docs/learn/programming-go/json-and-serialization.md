---
slug: json-and-serialization
title: JSON & serialization
description: "Turning Go structs into JSON and back with encoding/json — struct tags, exported fields, and the Marshal/Unmarshal pattern behind most APIs and config files."
keywords: go json, encoding/json, json marshal unmarshal, struct tags go, json tag, go serialization, exported fields json
level: intermediate
status: full
prereq:
  - structs-and-methods
faq:
  - q: Why is my struct field missing from the JSON output?
    a: "encoding/json only sees exported fields — ones that start with a capital letter. A lowercase field is unexported and invisible to the encoder, so it never appears in the output and is never filled in on decode. Capitalize it and, if you want a lowercase key, add a json struct tag."
  - q: What does a struct tag like `json:\"freq_hz\"` do?
    a: "It tells the encoder to use freq_hz as the JSON key for that field instead of the Go field name. Tags can also add options such as omitempty (drop the field when it's a zero value) and - (never encode this field)."
---

# JSON & serialization

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **`encoding/json`** package converts Go values to JSON with **`Marshal`** and
back with **`Unmarshal`**. Only **exported** (capitalized) fields are encoded.
**Struct tags** like `` `json:"freq_hz"` `` control the JSON key names and options
such as **`omitempty`**. This one pattern powers config files, HTTP APIs, and saved
state.
</div>

Serialization is how a program turns in-memory values into bytes it can store or send
— and back again. This lesson covers Go's built-in JSON support, building on
[Structs & methods](/learn/programming-go/structs-and-methods/).

## Marshal: struct to JSON

`json.Marshal` takes any value and returns its JSON bytes:

```go
type Channel struct {
    Name   string  `json:"name"`
    FreqHz float64 `json:"freq_hz"`
    Active bool    `json:"active"`
}

ch := Channel{Name: "Fire Dispatch", FreqHz: 460.025e6, Active: true}
data, err := json.Marshal(ch)
// {"name":"Fire Dispatch","freq_hz":460025000,"active":true}
```

The **struct tags** in backticks map each Go field to a JSON key. Without a tag the
field name is used verbatim (`FreqHz` would become `"FreqHz"`).

## Unmarshal: JSON back to a struct

`json.Unmarshal` fills a struct you pass by pointer:

```go
var ch Channel
err := json.Unmarshal(data, &ch)   // note &ch — it writes through the pointer
```

Fields present in the JSON are set; anything missing keeps its zero value. Extra keys
in the JSON that don't match a field are silently ignored, which makes forward-
compatible config easy.

## Exported fields only

The encoder can only see **exported** fields — the capitalization rule from
[Packages & modules](/learn/programming-go/packages-and-modules/). A lowercase field
is invisible:

```go
type Config struct {
    Gain    int    `json:"gain"`     // encoded
    secret  string                    // never encoded — unexported
}
```

This is the single most common JSON surprise in Go: a field "won't serialize" simply
because it starts with a lowercase letter.

## Tag options

Tags carry more than a name. The table lists the ones you'll reach for most.

| Tag | Effect |
|-----|--------|
| `json:"freq_hz"` | Use `freq_hz` as the key |
| `json:"name,omitempty"` | Omit the field when it's a zero value |
| `json:"-"` | Never encode or decode this field |
| `json:",omitempty"` (no name) | Keep the field name, add omitempty |

A GopherTrunk config that saves a scan list uses exactly this shape — a slice of
tagged structs marshalled to a file and unmarshalled on startup:

```go
type ScanList struct {
    Channels []Channel `json:"channels"`
    StepMS   int       `json:"step_ms,omitempty"`
}
```

For streaming large or continuous data, `json.NewEncoder(w)` and
`json.NewDecoder(r)` work directly on the `io.Writer`/`io.Reader` interfaces you'll
meet in the next lesson, instead of building one big byte slice.

<div class="knowledge-check" data-quiz data-correct-msg="Right — only exported (capitalized) fields are marshalled." markdown="0">
  <p class="knowledge-check__q">Quick check: why might a struct field never appear in json.Marshal output?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">It's unexported — its name starts with a lowercase letter</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's a float, which JSON can't represent</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Structs must be registered before marshalling</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`json.Marshal`** encodes a value; **`json.Unmarshal`** decodes into a pointer.
- Only **exported** fields are serialized — capitalization matters.
- **Struct tags** set JSON key names and options like **`omitempty`** and `-`.
- Encoders/decoders work over **`io.Reader`/`io.Writer`** for streaming.

Next up: reading and writing files and streams with os, io, and bufio.
