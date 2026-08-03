---
title: "The Operator's Cockpit, Part 13: The Reflect-Driven Config Form"
description: How one reflection walk over config.Config plus one shared field-metadata registry generates the whole config editor for both the terminal Bubbletea form and the web Config Builder — the TUI reading the registry directly, the web reading it over a fieldmeta endpoint, and two drift-guard tests keeping both editors honest to the struct.
category: deep-dives
keywords: reflect config form go, config builder schema, field metadata registry, one schema two editors, tui web config parity, reflection struct walk, hz freqlist widget, schema drift test, fieldmeta endpoint, gophertrunk operator cockpit
tags: [operator-cockpit, config, reflection, tui, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 13
---

*Part 13 of **The Operator's Cockpit**, the series on driving one GopherTrunk
daemon through one REST + SSE API from browser and terminal alike. Part 10 showed
how config edits are staged and activated safely; Part 12 showed the terminal
driving the same API as the browser. This post is the one place the two front-ends
share *more* than an API — a single reflection walk and a single metadata registry
generate the entire config editor for **both** the web Config Builder and the
terminal one.*

> **TL;DR:** GopherTrunk's config editor is generated, not hand-written. The
> terminal builder (`internal/configtui`) **walks `config.Config` with reflection**
> — every exported field becomes an editable row, its widget derived from the Go
> kind — so it *cannot* drift from the schema. The web builder is a hand-typed
> TypeScript form, but both editors read the **same field-metadata registry**
> (`internal/configbuilder`): labels, help text, select options, `Hz`/`FreqList`
> flags, and Advanced grouping, keyed `StructName.FieldName`. The TUI reads that
> registry directly; the web reads it over `GET /api/v1/config/fieldmeta`. Two
> tests hold the line: one fails if any config leaf lacks help, another fails if
> any config field is missing from the web's TypeScript schema.

**Key takeaways**

- **The TUI form is reflection, not markup.** `structRows` enumerates a struct's
  exported fields and `kindOf` picks a widget from the Go type — add a config
  field and it's editable in the terminal automatically.
- **One metadata registry, both editors.** Labels, help, options, and `Hz` /
  `FreqList` / Advanced flags live once in `configbuilder`; the TUI reads it in
  process, the web reads it over a fieldmeta endpoint.
- **Widgets are derived, overrides are declared.** The Go kind decides text vs
  number vs bool vs list; the registry's `Options`/`Hz`/`FreqList` *upgrade* a
  field to a select, a frequency box, or a freq-list.
- **Drift is a test failure, not a support ticket.** A coverage test rejects a new
  field with no help; a schema test rejects a new field the web form would
  silently drop — so both editors stay honest to the struct.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Reflection engine | struct → editable rows, widget per kind | `internal/configtui/reflectform.go` |
| Shared metadata | label/help/options/Hz/FreqList per field | `internal/configbuilder/fieldmeta.go` (`FieldMeta`) |
| TUI projection | reads the registry directly | `internal/configtui/meta.go` (`metaFor`) |
| Web fieldmeta API | serves the registry to the browser | `internal/api/handlers_config_builder.go` (`handleConfigFieldMeta`) |
| Path allow-list | validate load/save targets | `internal/api/config_builder.go` (`resolvePath`) |
| Coverage test | every leaf has help | `internal/configbuilder/fieldmeta_test.go` |
| Schema-drift test | every field covered by web types | `internal/configbuilder/webschema_test.go` |

## In this post

- **Why generate the form** — the cost of two hand-written config editors.
- **The reflection engine** — kinds to widgets, and the slice mutations.
- **The shared registry** — the one place polish lives.
- **Two editors, one source** — direct read vs the fieldmeta endpoint.
- **The drift guards** — two tests that keep both honest.

## Why generate the form

`config.Config` is large. It nests SDR sources, trunking systems with per-protocol
knobs, band plans, recording and enhancement chains, broadcast feeds, tone-out
profiles, retention, and more — hundreds of leaves. Hand-writing *one* editor for
that is a chore; hand-writing *two* (a terminal form and a web form) and keeping
them in sync as the schema grows is a standing liability. So the terminal builder
doesn't hand-write its form at all. It walks the struct:

```go
// internal/configtui/meta.go — package doc (shape)
// configtui is a standalone Bubble Tea Config Builder that mirrors the web
// Config Builder. It is reflection-driven: it walks config.Config so every
// field is editable automatically (it cannot drift from the schema), with a
// metadata table supplying the web-like polish — labels, help, select
// options, Hz formatting, fieldset grouping and the Advanced long-tail.
```

The phrase that matters is "it cannot drift from the schema." A reflection walk
sees whatever fields exist *now*; there's no second list to forget to update. Add
a field to a config struct and the terminal editor renders a row for it on the
next build — no wiring.

## The reflection engine

`structRows` is the heart of it: given a struct value, it enumerates the exported
fields, splits them into normal and Advanced (mirroring the web's grouping), and
builds one `formRow` per field with a widget kind chosen by `kindOf`:

```go
// internal/configtui/reflectform.go (shape)
func structRows(structName string, v reflect.Value) []formRow {
    for i := 0; i < v.Type().NumField(); i++ {
        f := v.Type().Field(i)
        if !f.IsExported() { continue }
        m := metaFor(structName, f.Name)      // shared registry lookup
        if m.Hidden { continue }
        r := formRow{
            field: f.Name, label: labelFor(structName, f), help: m.Help,
            advanced: isAdvanced(structName, f.Name), options: m.Options,
            kind: kindOf(f, m),                // Go type + meta → widget
        }
        // …normal fields first, then advanced
    }
}
```

`kindOf` is where a Go type becomes a UI widget. The base widget comes from the
`reflect.Kind`, and the registry *upgrades* it: a `string` with `Options` becomes
a select; a numeric with `Hz` becomes a frequency box; a `[]uint32` with
`FreqList` becomes a frequency-list widget; a `[]struct` becomes a drill-in list:

```go
// internal/configtui/reflectform.go (shape)
func kindOf(f reflect.StructField, m fieldMeta) rowKind {
    switch f.Type.Kind() {
    case reflect.String:
        if len(m.Options) > 0 { return kindSelect }
        return kindText
    case reflect.Int, /* …ints, uints, */ reflect.Float64:
        if m.Hz { return kindHz }
        if len(m.Options) > 0 { return kindSelect } // e.g. baud rendered as a choice
        return kindNumber
    case reflect.Slice:
        if m.FreqList { return kindFreqList }
        // []struct → kindList (drill in); []string → kindStringList;
        // []int → kindUnsupported (would corrupt on commit)
    case reflect.Ptr:
        // *bool → kindBoolPtr (tri-state); *struct → kindPtr (enable + drill)
    case reflect.Map:
        // map[string]bool (web.tabs) → kindMap; anything else → kindUnsupported
    }
    return kindUnsupported
}
```

The `kindUnsupported` fall-through is a deliberate safety valve — a `[]int` or an
exotic map isn't guessed at and mangled on commit; it's marked unsupported (and a
test guards which kinds are editable). Because rows are addressable reflect values,
editing a scalar sets it in place, and the list widgets get real mutation helpers
that operate on the addressable slice:

```go
// internal/configtui/reflectform.go (shape)
func appendItem(slice reflect.Value) int {        // grow by one, seed defaults
    elem := reflect.New(slice.Type().Elem()).Elem()
    applyNewDefaults(elem)                          // same defaults the web's makeNew uses
    slice.Set(reflect.Append(slice, elem)); return slice.Len()
}
func removeItem(slice reflect.Value, i int) { /* copy-down + reslice */ }
func dupItem(slice reflect.Value, i int)    { /* deep copy via JSON round-trip */ }
func moveItem(slice reflect.Value, i, delta int) { /* swap in place */ }
```

`applyNewDefaults` even matches the web builder's new-item defaults field-for-field
(a fresh `DeviceConfig` gets `Gain: "auto"`, a new `ConvChannelConfig` gets
`Mode: "nfm"`, a `EncryptionKeyConfig` gets `Algorithm: "rc4"`), so a row added in
the terminal is as valid out of the box as one added in the browser.

## The shared registry: where polish lives

Reflection gives you fields and types, but not *labels*, *help*, or the knowledge
that a `uint32` is really a frequency. That polish lives once, in a metadata
registry keyed by `StructName.FieldName`:

```go
// internal/configbuilder/fieldmeta.go (shape)
// FieldMeta is the shared, per-field metadata BOTH Config Builders consume: the
// terminal builder reads it directly; the web builder reads it over
// GET /api/v1/config/fieldmeta.
type FieldMeta struct {
    Label    string         `json:"label,omitempty"`
    Help     string         `json:"help,omitempty"`
    Options  []SelectOption `json:"options,omitempty"` // string/number → select
    Hz       bool           `json:"hz,omitempty"`      // render as MHz/Hz
    FreqList bool           `json:"freqList,omitempty"`// []uint32 → freq-list
}

var fieldMetas = map[string]FieldMeta{
    "SDRConfig.SampleRate":       {Label: "Sample rate", Hz: true, Help: "IQ rate every tuner is programmed to…"},
    "SystemConfig.Protocol":      {Help: "Trunking protocol this system uses.", Options: protocolOpts()},
    "SystemConfig.ControlChannels": {Label: "Control channels", FreqList: true, Help: "One or more control-channel frequencies…"},
    // …one entry per exported leaf, every one with non-empty Help
}
```

Every exported config leaf has an entry, and every entry carries help — that's the
comprehensive per-field text shown under the focused row in the terminal and beside
the widget in the browser. Because the registry is data, not code duplicated in two
front-ends, changing a help string or adding a select option updates *both* editors
at once.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="One schema and one metadata registry feed two config editors. config.Config is walked by reflection and combined with the shared FieldMeta registry. The terminal Config Builder reads the registry directly in process. The web Config Builder reads the same registry over GET /api/v1/config/fieldmeta. Both render the same labels, help, and widgets.">
  <rect x="250" y="14" width="160" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="31" text-anchor="middle" fill="var(--accent)" font-size="10">config.Config</text>
  <text x="330" y="45" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the one schema</text>
  <line x1="330" y1="52" x2="330" y2="74" stroke="currentColor"/><polygon points="326,74 330,84 334,74" fill="currentColor"/>
  <rect x="238" y="84" width="184" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="101" text-anchor="middle" fill="currentColor" font-size="10">reflection + FieldMeta registry</text>
  <text x="330" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="8">kinds → widgets · labels · help</text>
  <line x1="280" y1="124" x2="150" y2="160" stroke="currentColor"/><polygon points="150,156 141,162 153,164" fill="currentColor"/>
  <line x1="380" y1="124" x2="510" y2="160" stroke="currentColor"/><polygon points="510,164 519,162 507,156" fill="currentColor"/>
  <rect x="40" y="162" width="200" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="140" y="179" text-anchor="middle" fill="var(--accent)" font-size="10">terminal Config Builder</text>
  <text x="140" y="193" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reads registry directly</text>
  <rect x="420" y="162" width="200" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="520" y="179" text-anchor="middle" fill="var(--accent)" font-size="10">web Config Builder</text>
  <text x="520" y="193" text-anchor="middle" fill="var(--fg-muted)" font-size="8">GET /config/fieldmeta</text>
</svg>
<figcaption>The struct is the schema; the registry is the polish. Two editors, one source — the terminal reads it in process, the browser fetches it.</figcaption>
</figure>

## Two editors, one source

The terminal builder projects the shared registry into its own tiny view type,
straight from the package:

```go
// internal/configtui/meta.go (shape)
func metaFor(structName, field string) fieldMeta {
    fm := configbuilder.FieldMetaFor(structName, field) // the shared registry
    // …copy Label/Help/Hz/FreqList and map Options into the TUI's selOpt
}
```

The web builder can't import a Go package, so it fetches the identical data over
one endpoint:

```go
// internal/api/handlers_config_builder.go (shape)
// handleConfigFieldMeta answers GET /api/v1/config/fieldmeta — the shared
// per-field metadata table the web Config Builder renders labels/help from.
func (s *Server) handleConfigFieldMeta(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, configbuilder.FieldMetas())
}
```

Both editors save through the same allow-listed path resolver we met in Part 10 —
`resolvePath` symlink-resolves the target and confirms it lands inside a permitted
directory with a `.yaml`/`.yml` extension before anything is written — so "generate
the form" never becomes "let the form write anywhere." The generation is
convenient; the write is still fenced.

## The drift guards

Generating one editor from reflection and feeding a second from a shared registry
is only safe if something enforces that a schema change updates the metadata *and*
the web schema. Two tests do exactly that.

The **coverage test** (`fieldmeta_test.go`) fails if any exported config leaf lacks
a help string. As the `FieldMeta` comment puts it, "a schema change can't ship
without updating the shared help (and thus both editors)." A new field with no help
is a red build, not a blank tooltip discovered in production.

The **schema-drift test** (`webschema_test.go`) walks every config-package struct
reachable from `config.Config` and fails if a field has no counterpart in the web
builder's TypeScript schema and isn't on an explicit round-trip allow-list:

```go
// internal/configbuilder/webschema_test.go (shape)
// Fails when a config.Config field has no counterpart in the web builder's
// types.ts and is not allow-listed — i.e. a field the web Config Builder would
// silently drop on save/load.
func TestConfigSchemaCoveredByWebBuilder(t *testing.T) {
    fields := collectConfigStructFields()      // reflection over config.Config
    for structName, fnames := range fields {
        for fname := range fnames {
            if allow[structName][fname] { continue }
            if !typesDeclaresField(typesSrc, fname) {
                missing = append(missing, structName+"."+fname) // → test failure
            }
        }
    }
}
```

The allow-list is the escape hatch, and it's deliberately loud: a field that
round-trips through the web's generic index-signature editor (rather than a bespoke
typed widget) must be *recorded* there with a comment explaining why. So there's no
silent way to add a config field the web form would drop — you either give it a
typed editor or you consciously allow-list it as an index-signature round-trip. The
terminal editor, being reflection-driven, gets it for free. Together the two tests
turn "the config editors drifted from the struct" from a class of production bug
into a class of failing test.

## Where this goes next

[Part 14]({{ '/blog/deep-dives/operator-cockpit-14-testing-uis/' | relative_url }}),
the finale, pulls back to the question under all of this: how do you *test* a
browser and a terminal UI for a radio without a radio? API contract tests, TUI
model tests, and TypeScript unit tests that stand the whole cockpit up in memory —
the same pure-Go, in-memory discipline these drift guards are one example of.

## FAQ

**Is the terminal config form really generated by reflection?**
Yes. `internal/configtui` walks `config.Config` — every exported field becomes an
editable row, with the widget chosen from the Go kind. There's no hand-maintained
field list, which is precisely why the terminal editor can't drift from the schema.

**If it's reflection, where do labels and help come from?**
From a shared metadata registry in `internal/configbuilder`, keyed
`StructName.FieldName`. Reflection supplies the fields and types; the registry
supplies labels, help, select options, and the `Hz`/`FreqList`/Advanced flags that
upgrade a bare type into a richer widget.

**How does the web form get the same metadata?**
Over `GET /api/v1/config/fieldmeta`, which serves `configbuilder.FieldMetas()` —
the exact same registry the terminal reads in process. One source, two transports.

**What stops the two editors from drifting apart?**
Two tests. A coverage test fails if any config leaf lacks help (so both editors get
text). A schema-drift test fails if a config field isn't in the web's TypeScript
schema or the round-trip allow-list (so the web form can't silently drop a field).

**Why is the web form hand-typed but the TUI reflected?**
The TUI is Go, so it can reflect over `config.Config` at runtime. The web form is
TypeScript and can't import the Go struct, so it's hand-typed — but it consumes the
same metadata registry and is held to the struct by the schema-drift test.

## Series navigation

**Part 13 of 14** · ←
[Part 12: The TUI Cockpit — The Same API in Your Terminal]({{ '/blog/deep-dives/operator-cockpit-12-the-tui-cockpit/' | relative_url }})
· Next →
[Part 14: Testing Browser & Terminal UIs]({{ '/blog/deep-dives/operator-cockpit-14-testing-uis/' | relative_url }})
