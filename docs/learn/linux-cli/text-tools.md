---
slug: text-tools
title: Text-processing tools
description: The core Unix text filters — grep, sort, uniq, wc, cut, and a first taste of sed and awk — small single-purpose tools that snap together with pipes to turn raw text and logs into answers.
keywords: grep, sort, uniq, wc, cut, sed, awk, text processing, pipeline, log analysis, command line filters, count lines
level: intermediate
status: full
prereq:
  - pipes-and-redirection
faq:
  - q: "How do I count lines in a file?"
    a: "Use wc -l, like wc -l app.log to print the number of lines. Drop the -l and wc reports lines, words, and characters together. To count only the lines that match something, filter first and pipe into wc: grep ERROR app.log | wc -l."
  - q: "What does sort | uniq -c do?"
    a: "uniq only collapses adjacent duplicate lines, so you sort first to bring identical lines together, then uniq -c collapses each run into a single line prefixed with its count. The result is a tally of how many times each distinct line appeared."
  - q: "What's the difference between sed and awk?"
    a: "sed is a stream editor built for find-and-replace and line edits, as in sed 's/old/new/'. awk is a small programming language for column and field work, as in awk '{print $1}' to print the first field of every line. Both are deep tools; you can go a long way with just those two patterns."
  - q: "How do I pick one column out of a line?"
    a: "Use cut to select fields by a delimiter, like cut -d, -f2 to take the second comma-separated field, or awk '{print $2}' to take the second whitespace-separated field. cut is simplest for fixed delimiters; awk is more flexible when spacing varies."
---

# Text-processing tools

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`grep` / `sort` / `uniq` / `wc` / `cut`** are small single-purpose filters, and
**`sed` / `awk`** add stream editing and field work. Each does one thing to the text
flowing through it. Their power comes from snapping them together with
[pipes](/learn/linux-cli/pipes-and-redirection/): filter to the lines you want,
transform them, sort, then count. That pattern turns raw text and logs into answers
without writing a program.
</div>

The Unix philosophy is a drawer full of small tools that each do one job well and read
from a pipe. On their own they look almost too simple to be useful. Chained together,
they answer real questions — *how many errors today, which source is loudest, what
frequencies showed up* — in a single line.

## The filter toolkit

Each of these reads text (from a file or a pipe) and writes text, so any of them can
feed the next.

**`grep`** keeps only the lines that match. `grep "error" app.log` prints every line
containing `error`; `-i` ignores case and `-v` inverts the match to *exclude* lines.
It's usually the first stage of a pipeline — narrow the flood to what you care about.

**`sort`** puts lines in order. Alphabetical by default; `-n` sorts numerically and
`-r` reverses. Beyond tidiness, sorting **groups identical lines together**, which is
what makes the next tool work.

**`uniq`** collapses **adjacent** duplicate lines into one. Because it only looks at
neighbours, you almost always `sort` first. The killer flag is `-c`, which prefixes
each line with how many times it occurred — an instant tally.

**`wc`** counts. `wc -l` gives lines, `-w` words, `-c` characters; with no flag you get
all three. Piped onto the end of a filter, `wc -l` answers "how many?".

**`cut`** picks columns out of each line. `cut -d, -f2` takes the second
comma-separated field; `-f1,3` takes the first and third. Good for pulling one column
out of structured text like CSV or log lines with a fixed separator.

**`head`** and **`tail`** trim to the top or bottom N lines (`-n 20`). `head` caps a
long result; `tail -f` follows a file live as new lines are written — handy for
watching a log.

## A taste of sed and awk

Two older tools go further, and both deserve a lesson of their own — this is just a
first look.

**`sed`** is a *stream editor*: it edits text as it flows past. The one pattern worth
knowing now is substitution:

```
sed 's/old/new/'
```

That replaces the first `old` with `new` on each line. Add a trailing `g` (`s/old/new/g`)
to replace every occurrence. It's find-and-replace for a pipe, no file opened.

**`awk`** treats each line as a set of fields and lets you work with them by number:

```
awk '{print $1}'
```

`$1` is the first whitespace-separated field, `$2` the second, and so on. That one line
plucks a column out of ragged text where `cut` struggles with uneven spacing. awk is a
whole small language — arithmetic, conditions, running totals — but even `{print $N}`
earns its place in your toolkit.

## Building pipelines

The recurring pattern is **filter → transform → sort → count**: cut the input down with
`grep`, reshape it with `cut`/`sed`/`awk`, `sort` to group, then `uniq -c` or `wc -l` to
tally. Read a pipeline left to right as a sentence.

Count how many `ERROR` lines a log holds:

```
grep ERROR app.log | wc -l
```

`grep` keeps only the error lines, `wc -l` counts them. Now find the **busiest sources**
— which value appears most often — assuming it sits in the first field:

```
cut -d' ' -f1 app.log | sort | uniq -c | sort -rn | head
```

Read it as: pull the first field, sort so identical values sit together, `uniq -c` to
tally each, `sort -rn` to put the biggest counts on top, `head` to show the top few.
That "sort | uniq -c | sort -rn" run is the classic top-N idiom you'll reuse constantly.

## Applied to GopherTrunk

A scanner's logs are just text, so the same moves apply. To count how often a particular
talkgroup shows up:

```
grep "TG 12345" scanner.log | wc -l
```

Or to see **which** talkgroups were most active across a session — extract the field,
then run the top-N idiom:

```
grep "TG " scanner.log | awk '{print $3}' | sort | uniq -c | sort -rn | head
```

The same shape extracts the **frequencies seen**: `grep` the lines, `cut` or `awk` the
frequency column, `sort | uniq -c` to tally. A long, noisy log becomes a short summary —
no database, no script, just filters and pipes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — sort groups identical lines, then uniq -c tallies each one." markdown="0">
  <p class="knowledge-check__q">Quick check: what does <code>sort | uniq -c</code> produce from a list of lines?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The lines with all duplicates removed and nothing else</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A count of each unique line</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The lines reversed alphabetically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`grep`** keeps matching lines, **`sort`** orders them, **`uniq -c`** tallies adjacent duplicates, **`wc`** counts, **`cut`** picks columns.
- **`sort` before `uniq`** — uniq only collapses *adjacent* lines.
- **`sed 's/old/new/'`** does find-and-replace in a stream; **`awk '{print $1}'`** pulls fields — both are deep tools you're only sampling here.
- The pipeline pattern is **filter → transform → sort → count**; `sort | uniq -c | sort -rn | head` is the top-N idiom.
- The same moves turn a scanner's logs into summaries — counting talkgroups or extracting frequencies with a single chain of filters.

Next up: [wildcards, globbing & expansion](/learn/linux-cli/wildcards-and-expansion/)
