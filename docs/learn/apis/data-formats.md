---
slug: data-formats
title: "Data formats: JSON and friends"
description: How structured data survives the trip between programs — JSON's syntax and types, where YAML and CSV fit, and why binary encodings exist. The serialization ideas every API request and response depend on.
keywords: JSON tutorial, data serialization, JSON vs YAML, JSON vs XML, binary encoding, data formats for APIs, structured data
level: beginner
status: full
---

# Data formats: JSON and friends

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Programs keep data in memory as structures; networks carry only bytes. **Serialization**
turns structures into bytes and back, and a **data format** is the agreed way to do it.
**JSON** is the lingua franca of web APIs — human-readable text with objects, arrays,
strings, numbers, and booleans. **YAML** trades strictness for readability and rules
config files; **CSV** rules tabular exports; **binary formats** like Protocol Buffers
trade readability for compactness and speed. Choosing a format is choosing who has to
be able to read it.
</div>

An API contract is mostly a promise about *data*: send a request shaped like this,
get a response shaped like that. This lesson is about the shapes themselves — how
structured data is written down so it survives the trip between two programs that
share nothing but a byte stream.

## Why does data need a format at all?

Inside a running program, "a call record" is a structure in memory — fields, types,
pointers. None of that survives a network hop: the wire carries bytes, and the
program on the far end has its own memory layout, possibly its own language.
**Serialization** flattens the structure into an agreed byte sequence;
**deserialization** (parsing) rebuilds it on the other side. The format is the
protocol-within-the-protocol: both sides must agree on it or the data is noise —
the same lesson as [What is a protocol?](/learn/apis/what-is-a-protocol/), one
layer up.

## JSON — the web's default

**JSON** (JavaScript Object Notation) is how the overwhelming majority of web APIs
write their data. A call record from a scanner daemon might look like:

```json
{
  "id": 48213,
  "system": "county-p25",
  "talkgroup": 1201,
  "label": "County Fire Dispatch",
  "start": "2026-08-21T14:03:12Z",
  "duration_seconds": 8.4,
  "encrypted": false,
  "units": [70233, 70281]
}
```

JSON has exactly six kinds of value, and you just saw them all: **objects**
(`{"key": value}` pairs), **arrays** (`[...]`), **strings**, **numbers**,
**booleans**, and `null`. That tiny vocabulary is a feature: every language can
parse it, humans can read it, and there's little room for dialects. Note what JSON
*doesn't* have: no comments, no dates (they ride as strings, usually ISO 8601 like
the `start` above), and no distinction between integers and floats — conventions
fill those gaps, and a good API documents its conventions.

If you write Go, the
[JSON & serialization lesson](/learn/programming-go/json-and-serialization/) shows
how structs map to JSON with tags; here it's enough to read it fluently.

## YAML, CSV, XML — the supporting cast

| Format | Best at | Watch out for |
|--------|---------|---------------|
| **JSON** | API requests/responses, anything program-to-program | No comments; numbers lose precision past 2⁵³ |
| **YAML** | Config files humans edit (GopherTrunk's own config is YAML) | Whitespace-sensitive; surprising type coercions |
| **CSV** | Tabular exports — talkgroup lists, call logs into spreadsheets | No nesting, no types; quoting edge cases |
| **XML** | Legacy enterprise APIs, document markup | Verbose; mostly displaced by JSON for new APIs |

The pattern to internalise: **YAML for humans writing, JSON for programs talking,
CSV for tables**. GopherTrunk follows it exactly — a YAML config file you edit, a
JSON API the console consumes, CSV talkgroup files you can import from community
databases.

## When text isn't enough: binary formats

Text formats spend bytes generously — the number `48213` costs five bytes as JSON
digits but four (or fewer) as a binary integer, and every field name is spelled out
in full in every single record. For an API returning ten call records, nobody
cares. For a stream of thousands of messages a second — or audio samples — the
overhead is real, in bandwidth and in parsing time.

**Binary formats** like Protocol Buffers write data as compact, typed bytes with
the field names factored out into a shared schema. The cost is that you can no
longer read a message with your eyes or a text editor — you need the schema and
tooling. That trade gets a full treatment in
[text vs binary protocols](/learn/apis/text-vs-binary-protocols/) and
[gRPC & Protocol Buffers](/learn/apis/grpc-and-protobuf/); for now, know both
families exist and why.

> Rule of thumb: start with JSON. Reach for binary when measurement — not
> intuition — says the encoding is your bottleneck.

## How do both sides stay in sync?

A format only fixes the *syntax*. Both sides must still agree that `talkgroup` is a
number and `label` is a string — the *schema*. With JSON that agreement often lives
only in documentation and habit, which is exactly where contracts quietly rot: a
server starts sending `duration` instead of `duration_seconds`, and a client
somewhere breaks. Formal schemas and generated code close that gap, and they're the
subject of [schemas & code generation](/learn/apis/schemas-and-codegen/). The next
lesson makes the promise itself — the contract — precise.

<div class="knowledge-check" data-quiz data-correct-msg="Right — serialization is flattening in-memory structures into an agreed byte format for the trip." markdown="0">
  <p class="knowledge-check__q">Quick check: what does it mean to serialize a data structure?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To encrypt it so only the recipient can read it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To compress it so it takes fewer bytes on the wire</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To convert it into an agreed byte format that the other side can parse back</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Networks carry bytes, not structures — **serialization** and a shared **data
  format** are what let structured data cross between programs.
- **JSON** is the default for web APIs: six value types, readable by humans and
  parseable by everything.
- **YAML** suits human-edited config, **CSV** suits tables, **XML** lingers in
  legacy APIs.
- **Binary formats** trade human readability for compactness and speed — worth it
  for high-volume streams, overkill for most requests.
- A format fixes syntax only; the field-level agreement is the **schema**, and
  keeping both sides in sync is the contract problem the next lesson tackles.

Next up: [API contracts](/learn/apis/api-contracts/).
