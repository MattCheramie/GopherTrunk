---
slug: conditionals-and-loops
title: Conditionals & loops
description: How a shell script makes decisions and repeats work — if/then/elif/else, the test command and its [ ] and [[ ]] forms for file, string and number tests, for loops over lists and globs, while loops and reading a file line by line, and why conditions are really exit codes.
keywords: bash if, for loop, while loop, test command, conditionals, file test, string test, control flow, elif, exit code, glob loop, shell scripting
level: intermediate
status: full
prereq:
  - first-shell-script
faq:
  - q: "How does an if statement work in bash?"
    a: "An if statement runs a command and checks whether it succeeded. The shape is if COMMAND; then ...; fi — if COMMAND returns exit code 0 (success) the then block runs, otherwise it is skipped. Add elif for more conditions and else for a fallback, and close the block with fi. Most often the command you test is the test command, written as [ ... ], which checks files, strings and numbers."
  - q: "What is the difference between [ ] and [[ ]] in bash?"
    a: "Both test a condition. Single [ ] is the classic test command and works in every POSIX shell, but you must quote variables carefully because unquoted empty values break it. Double [[ ]] is a bash builtin that is safer with unquoted variables and adds pattern matching and && / || inside the brackets. Use [[ ]] in bash scripts; use [ ] when the script must run under plain sh."
  - q: "How do I loop over files in a directory?"
    a: "Use a for loop over a glob: for f in *.log; do ...; done. The shell expands *.log to the matching filenames before the loop runs, and f holds each name in turn. Always quote the variable inside the loop (\"$f\") so filenames with spaces are handled as one item, and remember that if nothing matches the pattern is left as the literal text unless nullglob is set."
  - q: "How do I read a file line by line in a script?"
    a: "Use a while loop with read: while read -r line; do ...; done < file.txt. The read command pulls one line at a time into the variable line, and the loop ends when read reaches end of file. The -r flag stops backslashes being treated specially, and redirecting the file in with < feeds it as the loop's input."
---

# Conditionals & loops

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Control flow is what turns a list of commands into a program that can *think*.
**`if` / `test [ ]`** lets a script make a decision — run this block only when a
condition holds. **`for`** repeats a block once per item in a list or glob, and
**`while`** repeats as long as a condition stays true. Together they let a
[script](/learn/linux-cli/first-shell-script/) decide and repeat instead of
running the same lines blindly from top to bottom.
</div>

A plain script runs every line in order, no matter what. That is fine until you
need it to *react* — back up a file only if it exists, process every capture in a
folder, keep retrying until something comes up. Those three needs map onto the
three tools in this lesson: decisions, and two kinds of repetition.

## Making decisions — if

An `if` statement runs a block only when a condition is met. The shape is
always the same, and it closes with `fi` (`if` backwards):

```
if [ -f config.toml ]; then
  echo "found the config"
elif [ -f config.yaml ]; then
  echo "found a YAML config instead"
else
  echo "no config here"
fi
```

Read it top to bottom: the first condition that holds wins, its `then` block
runs, and the rest are skipped. `elif` ("else if") lets you test more
conditions, and `else` is the catch-all when none matched. Only `if` and `fi`
are required — `elif` and `else` are optional.

## The test command

The thing between the brackets is the **test command**. `[ ... ]` is actually a
program (its condition goes inside), which is why it needs spaces around the
brackets. In bash you can also write `[[ ... ]]`, a safer builtin that copes with
unquoted variables and adds pattern matching — prefer it in bash scripts, and
keep `[ ]` for scripts that must run under plain `sh`.

Test comes in three flavours — files, strings, and numbers:

| Test | True when |
|------|-----------|
| `[ -f path ]` | `path` exists and is a regular **file** |
| `[ -d path ]` | `path` exists and is a **directory** |
| `[ -e path ]` | `path` **exists** (any type) |
| `[ -z "$s" ]` | string `$s` is **empty** (zero length) |
| `[ "$a" = "$b" ]` | strings are **equal** |
| `[ "$a" != "$b" ]` | strings are **not equal** |
| `[ "$n" -eq 5 ]` | number **equals** 5 |
| `[ "$n" -lt 5 ]` | number is **less than** 5 |
| `[ "$n" -gt 5 ]` | number is **greater than** 5 |

Note the split: strings compare with `=` and `!=`, but numbers use the word
operators `-eq`, `-lt`, `-gt`. Always quote your variables (`"$s"`) so an empty
value does not break the brackets.

## for loops

A `for` loop runs its block once for each item in a list. The classic use is
looping over a **glob** — the shell expands the pattern to matching filenames
first, then hands them to the loop one at a time:

```
for f in *.log; do
  echo "checking $f"
done
```

The list can be anything, not just files — spell it out inline, or loop over the
script's own **arguments** with `"$@"`:

```
for name in alpha bravo charlie; do
  echo "$name"
done

for arg in "$@"; do
  echo "you passed: $arg"
done
```

Quote the loop variable (`"$f"`) so a filename with spaces stays a single item.

## while loops

A `while` loop repeats as long as its condition stays true — useful when you do
not know the count in advance:

```
n=1
while [ "$n" -le 3 ]; do
  echo "attempt $n"
  n=$((n + 1))
done
```

The most common real use is reading a file **line by line**, pairing `while`
with the `read` command:

```
while read -r line; do
  echo "got: $line"
done < channels.txt
```

`read` pulls one line into `line` each pass and the loop ends at end of file.
The `-r` flag stops backslashes being mangled, and `< channels.txt` feeds the
file in as input.

## A worked example

Here is control flow doing something real: back up every `.conf` file in the
current folder, but only the ones that actually exist as regular files.

```
#!/usr/bin/env bash
for f in *.conf; do
  if [ -f "$f" ]; then
    cp "$f" "$f.bak"
    echo "backed up $f"
  fi
done
```

The `for` loop walks each `.conf` name; the `if [ -f "$f" ]` guard means a
pattern that matched nothing (left as the literal `*.conf`) is skipped rather
than copied by mistake. Decision and repetition together — the whole point of
this lesson in five lines.

## Conditions are exit codes

Here is the idea that ties it together: `if` does not test *true or false*, it
tests whether a **command succeeded**. Every command reports an **exit code** —
`0` for success, non-zero for failure — and `if` runs its `then` block when the
code is `0`. `[ ... ]` is just a command that exits `0` when its condition holds.

That means you can test *any* command, not only `[ ]`:

```
if grep -q ERROR log.txt; then
  echo "the log has errors"
fi
```

`grep -q` prints nothing but exits `0` when it finds a match — perfect for an
`if`. Exit codes are the currency of shell control flow, and the next lesson on
[functions, exit codes & error handling](/learn/linux-cli/functions-and-exit-codes/)
digs into them properly.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the glob expands to the .log files and the loop runs once per name." markdown="0">
  <p class="knowledge-check__q">Quick check: which builds a loop that runs once per <code>.log</code> file in the folder?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>if [ -f *.log ]; then ... fi</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="correct"><code>for f in *.log; do ... ; done</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>while *.log; do ... ; done</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`if ... then ... fi`** runs a block only when a condition holds; add `elif`
  and `else` for more branches.
- The **test command** `[ ... ]` (and bash's safer `[[ ... ]]`) checks files
  (`-f`, `-d`, `-e`), strings (`-z`, `=`, `!=`), and numbers (`-eq`, `-lt`, `-gt`).
- **`for`** repeats once per item in a list or glob; loop over arguments with `"$@"`.
- **`while`** repeats while a condition holds — including reading a file with
  `while read -r line`.
- Quote your variables (`"$f"`) so spaces and empty values do not break things.
- `if` really tests a command's **exit code** — `0` means success.

Next up: [functions, exit codes & error handling](/learn/linux-cli/functions-and-exit-codes/)
