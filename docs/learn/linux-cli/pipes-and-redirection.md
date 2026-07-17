---
slug: pipes-and-redirection
title: Pipes & redirection
description: How the shell wires programs together — the three standard streams stdin, stdout and stderr, redirecting output to files with > and >>, feeding files in with <, sending errors with 2>, and chaining tools with the pipe | to build powerful one-line pipelines.
keywords: pipe, redirection, stdin, stdout, stderr, ">", ">>", "<", "2>", unix philosophy, shell pipeline, grep, command line
level: intermediate
status: full
prereq:
  - first-commands
faq:
  - q: What is the difference between a pipe and redirection?
    a: "Redirection connects a command to a file: > and >> write its output to a file, and < reads a file as its input. A pipe (|) connects two commands directly, sending the first command's output straight into the second's input without any file in between. Rule of thumb: use redirection to talk to files, use a pipe to talk to another program."
  - q: What are stdin, stdout and stderr?
    a: "They are the three standard streams every command has. Standard input (stdin) is where it reads from — your keyboard by default. Standard output (stdout) is where normal results go — your screen by default. Standard error (stderr) is a separate channel for error and warning messages, also the screen by default. Keeping errors on their own stream lets you redirect results and error messages to different places."
  - q: What is the difference between > and >>?
    a: "Both send stdout to a file. A single > overwrites the file, replacing whatever was there — so it is easy to wipe a file by accident. A double >> appends, adding to the end and keeping the existing contents. When in doubt, reach for >> so you do not clobber something you wanted to keep."
  - q: How do I save error messages to a file?
    a: "Use 2> to redirect the error stream, for example cmd 2> errors.log. That sends stderr to the file while normal output still goes to your screen. To capture both normal output and errors in one file, use &> (as in cmd &> all.log), which redirects stdout and stderr together."
---

# Pipes & redirection

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every command has three streams: **stdin** (input), **stdout** (normal output),
and **stderr** (errors). Redirection wires those streams to files — **`>`**
overwrites, **`>>`** appends, and **`<`** feeds a file in as input. The
**pipe `|`** wires one command's output straight into the next command's input.
This is where the shell stops being a place to type single
[commands](/learn/linux-cli/first-commands/) and becomes a place to *build* things:
chain small tools together and send their output to and from files.
</div>

Up to now each command has stood alone: you type it, it prints to the screen, done.
The real power of the shell is connecting commands to each other and to files. Once
this clicks, a single line can answer questions that would take a whole program
otherwise.

## Three streams

Every command you run has three standard streams attached to it:

- **standard input** (*stdin*) — where the command reads from.
- **standard output** (*stdout*) — where its normal results go.
- **standard error** (*stderr*) — a separate channel for error and warning messages.

By default, **input comes from your keyboard** and **both output streams go to your
screen**. That is why a command just prints its results and you just type at it —
the streams are pointed at you. Redirection and pipes are nothing more than pointing
those streams somewhere else.

Errors travelling on their own stream is the clever part: it means you can capture a
command's results in a file while its error messages still appear on screen, or send
each to a different place.

## Redirecting output

The **`>`** operator sends stdout to a file instead of the screen. It **overwrites**
the file — whatever was there is gone:

```
ls > files.txt        # save the listing to files.txt (replaces it)
```

Use **`>>`** to **append** instead, adding to the end and keeping the existing
contents:

```
echo "run started" >> log.txt   # add a line without wiping the file
```

Errors ride the separate stderr stream, so **`>`** does not capture them. Redirect
those with **`2>`** (the `2` is stderr's stream number):

```
find / -name '*.cfile' 2> errors.log   # results to screen, errors to a file
```

To capture normal output *and* errors together in one file, use **`&>`**:

```
make test &> build.log          # everything, stdout and stderr, into build.log
```

## Redirecting input

The mirror image of `>` is **`<`**, which feeds a file in as a command's stdin —
as if you had typed the file's contents at the keyboard:

```
sort < names.txt      # sort reads its input from the file
```

You will reach for `<` less often than the others, because most tools also accept a
filename directly, but it is worth recognising when you see it.

## Pipes

The **pipe `|`** is the star. It connects one command's **stdout** directly to the
next command's **stdin** — no file in between. The output of the left-hand command
becomes the input of the right-hand command:

```
ls | less             # page through a long listing instead of it scrolling past
cat log.txt | grep ERROR   # keep only the lines containing ERROR
```

You can chain as many stages as you like, each one transforming what the last handed
it. That is how you *compose* small tools into something bigger without writing a
single line of code.

## The Unix philosophy

There is a design idea behind all of this: **write small programs that each do one
thing well, then combine them with pipes**. Rather than one giant program with a
setting for everything, Unix gives you a toolbox of sharp little tools — one lists
files, one filters lines, one counts, one sorts — and the pipe lets you snap them
together for the job in front of you.

This is the shell's real superpower, and it is why the command line stays useful for
tasks nobody anticipated. The next lesson on
[text-processing tools](/learn/linux-cli/text-tools/) is where those sharp little
tools really pay off.

## Worked examples

A pipeline is best understood by reading it left to right, one stage at a time.
Count how many lines in a log mention an error:

```
cat log.txt | grep ERROR | wc -l
```

Read it as a sentence: `cat` pours the file into the pipe, `grep ERROR` keeps only
the matching lines, and `wc -l` counts what survives. The screen shows a single
number — the answer to "how many errors?".

Here is another, finding which processes are using the most of something without
scrolling by hand:

```
ps aux | grep gophertrunk
```

`ps aux` lists every running process; `grep gophertrunk` throws away every line that
does not mention it. You asked a real question and answered it in one line — and each
piece was a tool you already knew.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the pipe feeds ls's output into grep, which keeps only matching lines." markdown="0">
  <p class="knowledge-check__q">Quick check: what does <code>ls | grep .txt</code> do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Renames every file to end in .txt</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Sends ls's output into grep, showing only lines containing .txt</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Deletes all the .txt files in the directory</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Every command has three streams: **stdin**, **stdout**, and **stderr**.
- **`>`** writes stdout to a file (overwrites); **`>>`** appends to it.
- **`2>`** redirects errors; **`&>`** captures both output and errors together.
- **`<`** feeds a file in as a command's input.
- The **pipe `|`** connects one command's output straight into the next's input.
- Small tools chained with pipes — the **Unix philosophy** — is the shell's real power.

Next up: [text-processing tools](/learn/linux-cli/text-tools/)
