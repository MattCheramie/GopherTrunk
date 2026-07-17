---
slug: variables-and-input
title: Variables, arguments & input
description: How to make a shell script flexible — storing values in variables with NAME=value, reading them back with $NAME, taking positional arguments like $1 and $@, prompting the user with read, capturing command output with $(command), and why you should always quote your variables.
keywords: bash variables, script arguments, $1, positional parameters, read input, command substitution, quoting, shell script variables, $@, default value
level: intermediate
status: full
prereq:
  - first-shell-script
faq:
  - q: "How do I use a variable in a bash script?"
    a: "Assign it with NAME=value and no spaces around the equals sign, then read it back by putting a dollar sign in front: $NAME or ${NAME}. The variable lives inside the running script. Always wrap it in double quotes when you use it — \"$NAME\" — so values containing spaces stay in one piece."
  - q: "What is $1 in a shell script?"
    a: "$1 is the first argument passed to the script on the command line — a positional parameter. $2 is the second, and so on. $0 is the script's own name, $@ is all the arguments together, and $# is how many arguments were given. They let a script act on whatever the user types after its name."
  - q: "How do I read user input in bash?"
    a: "Use the read command: read NAME pauses the script and stores whatever the user types into the variable NAME. read -p \"Prompt: \" NAME prints a prompt on the same line first. It's how a script asks a question interactively instead of relying only on command-line arguments."
  - q: "What is command substitution in bash?"
    a: "Command substitution runs a command and captures its output as text you can store or use. Write it as $(command) — for example today=$(date +%F) puts today's date into the variable today. The older backtick form `command` does the same thing but nests badly, so $() is preferred."
---

# Variables, arguments & input

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three things turn a hard-coded script into a flexible one: **variables**
(`NAME=value`, read back as **`$NAME`**), **arguments** (**`$1`**, **`$@`** — what
the user types after the script name), and **`read`** (asking the user a question
interactively). And one rule that saves hours of debugging: **always quote your
variables** — `"$NAME"`. If scripts are new to you, start with
[your first shell script](/learn/linux-cli/first-shell-script/).
</div>

A script that always does exactly the same thing is barely more than a note to
yourself. The moment you want it to act on *a file you name*, *a value you type*,
or *today's date*, you need variables, arguments, and input. They are what let one
script serve a hundred situations.

## Variables

A variable is a name with a value attached. Create one by writing the name, an
equals sign, and the value — **no spaces around the `=`**:

```
name=Matt
```

Read it back by putting a `$` in front. Use `${name}` with braces when the name
butts up against other text:

```
echo "$name"
echo "${name}_backup"
```

The spaces rule is strict: `name = Matt` (with spaces) is *not* an assignment — the
shell reads it as a command called `name`. Keep the `=` tight.

A variable made this way is **local to the running script**: it lives while the
script runs and is gone when it finishes. (To hand a value down to programs the
script *launches*, you'd export it — see
[environment variables](/learn/linux-cli/environment-variables/).)

One habit to build now: **always quote a variable when you use it** —
`"$name"`, not `$name`. Quoting keeps a value that contains spaces in one piece
and stops the shell from trying to expand it. The
[wildcards & expansion](/learn/linux-cli/wildcards-and-expansion/) lesson shows
exactly what goes wrong when you don't.

## Script arguments

When you run a script, anything you type after its name is passed in as
**positional parameters**. Inside the script they arrive as numbered variables:

- **`$1`**, **`$2`**, `$3` … — the first, second, third argument, and so on.
- **`$0`** — the name the script was invoked as.
- **`$@`** — all the arguments together.
- **`$#`** — the *count* of arguments.

Here's a script that greets whoever you name on the command line:

```bash
#!/usr/bin/env bash
echo "Hello, $1! You passed $# argument(s)."
```

Save it as `greet.sh`, make it executable, and run it:

```
./greet.sh world
```

That prints `Hello, world! You passed 1 argument(s).` — `$1` picked up `world`.
Run it with no arguments and `$1` is simply empty, which is exactly the case the
next section handles.

## Reading input

Arguments come from the command line; **`read`** asks the user *while the script is
running*. `read` pauses, waits for a line of typing, and stores it in a variable:

```bash
#!/usr/bin/env bash
read name
echo "Hello, $name!"
```

To print a prompt on the same line first, use **`read -p`**:

```bash
read -p "What is your name? " name
echo "Hello, $name!"
```

The script stops at the `read`, shows `What is your name? `, and waits. Whatever
the user types lands in `name`.

## Command substitution

Often the value you want *is the output of another command*. **Command
substitution** runs a command and captures its output as text. Write it as
**`$(command)`**:

```bash
today=$(date +%F)
echo "Today is $today"
```

`date +%F` prints something like `2026-07-17`, and `$( )` catches that text and
stores it in `today`. You can drop a substitution anywhere a value is expected:

```
echo "There are $(ls | wc -l) items here."
```

An older form uses backticks — `` today=`date +%F` `` — and does the same thing,
but backticks nest awkwardly and are easy to misread, so **`$( )` is the modern
default**.

## A small useful script

Put the pieces together: take a name as an argument, fall back to a **default** if
none is given, and stamp the greeting with today's date via command substitution:

```bash
#!/usr/bin/env bash
# Greet someone, defaulting to "friend" if no name is given.
name="${1:-friend}"
today=$(date +%F)
echo "Hello, $name — today is $today."
```

The trick is **`${1:-friend}`**: "use `$1`, but if it's unset or empty, use
`friend` instead." Run `./greet.sh Matt` and it greets Matt; run it with no
argument and it greets `friend`. One script, both cases handled.

## Quoting matters

This is the mistake that bites everyone eventually. An **unquoted** variable is
split on spaces and then checked for wildcards before your command ever sees it.
Say a file is called `my notes.txt`:

```
file="my notes.txt"
rm $file      # WRONG: runs rm "my" "notes.txt" — two files!
rm "$file"    # RIGHT: one argument, the real filename
```

Unquoted, the space splits `$file` into two arguments and `rm` tries to delete
`my` and `notes.txt`. Quoted, it stays one filename. The rule is simple:
**double-quote your variables by default** — `"$file"`, `"$1"`, `"$name"` — and
only leave the quotes off when you have a specific reason. The same splitting and
globbing behaviour is explained in full under
[environment variables](/learn/linux-cli/environment-variables/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — $1 holds the first argument passed on the command line." markdown="0">
  <p class="knowledge-check__q">Quick check: inside a script, which variable holds the first argument passed on the command line?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">$0</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">$1</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">$#</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Set a variable with **`NAME=value`** (no spaces around `=`) and read it with **`$NAME`** or **`${NAME}`**; it's local to the running script.
- Command-line arguments arrive as **positional parameters**: **`$1`**, **`$2`**, `$0` (the script name), **`$@`** (all of them), **`$#`** (the count).
- **`read VAR`** prompts the user interactively; **`read -p "…"`** shows a prompt first.
- **`$(command)`** captures a command's output into a value — e.g. `today=$(date +%F)`.
- **`${1:-default}`** supplies a fallback when an argument is missing.
- **Always double-quote your variables** — `"$NAME"` — so values with spaces survive and don't get globbed.

Next up: [conditionals & loops](/learn/linux-cli/conditionals-and-loops/)
