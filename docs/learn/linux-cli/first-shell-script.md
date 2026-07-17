---
slug: first-shell-script
title: Your first shell script
description: How to write and run your first shell script — saving a sequence of commands in a file, the shebang line that picks the interpreter, making the file executable with chmod +x, running it with ./script.sh, adding comments, and why scripting turns anything you type twice into one command.
keywords: shell script, bash script, shebang, chmod +x, automation, run script, executable, script file, bin bash, env bash
level: intermediate
status: full
prereq:
  - pipes-and-redirection
faq:
  - q: What is a shell script?
    a: "A shell script is just a text file containing a list of shell commands. When you run the file, the shell reads it top to bottom and executes each line exactly as if you had typed it at the prompt. It lets you save a sequence of commands once and run the whole thing as a single program."
  - q: What does the shebang line do?
    a: "The shebang is the first line of the script, like #!/bin/bash or #!/usr/bin/env bash. The #! tells the system which interpreter should run the file. Without it, the system may not know whether to use bash, another shell, or some other language, so the same script can behave differently or fail to run."
  - q: What does chmod +x do to a script?
    a: "chmod +x adds the execute permission bit to the file, marking it as a program you are allowed to run. Until you set that bit, the shell refuses to run the file directly with ./script.sh. Alternatively you can skip the bit and run it as bash script.sh, which tells bash to read the file itself."
  - q: Why do I have to type ./ before my script name?
    a: "The ./ tells the shell to run the script in the current directory. The shell only searches the directories listed in your PATH for bare command names, and the current directory is normally not on PATH for safety. Writing ./script.sh gives an explicit location so the shell knows exactly which file to run."
---

# Your first shell script

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **script** is just a file full of commands you can run as one program. Put a
**shebang `#!`** on the first line so the system knows which interpreter to use,
mark the file runnable with **`chmod +x`**, then run it with `./script.sh`.
This is the start of automating anything you find yourself typing more than once —
the natural next step after wiring commands together with
[pipes and redirection](/learn/linux-cli/pipes-and-redirection/).
</div>

Up to now every command has been something you type by hand, one line at a time. A
script lets you write that sequence down once and replay it whenever you like. The
mechanics are tiny — three ideas — and once they click you can automate almost
anything you do at the shell.

## What a script is

A shell script is nothing more than a **text file with commands in it**. You save the
same lines you would type at the prompt, and when you run the file the shell reads it
from top to bottom and **executes each line as if you had typed it yourself**.

That is the whole concept. There is no special scripting language to learn first — you
already know the language, because it is the same commands you have been using. A
script just collects them in one place so you can run them together.

## The shebang line

The **first line** of a script is special. If it begins with `#!` — called the
**shebang** — the rest of that line names the **interpreter** that should run the file:

```
#!/bin/bash
```

You will also often see this form:

```
#!/usr/bin/env bash
```

Both point the system at `bash`. The difference is that `#!/bin/bash` uses bash at a
fixed location, while `#!/usr/bin/env bash` asks `env` to find bash on your
[PATH](/learn/linux-cli/environment-variables/) — handier on systems where bash lives
somewhere unusual.

Why it matters: without a shebang the system has to guess which interpreter to use, and
the same file can behave differently or refuse to run. The shebang removes the guess —
the script says, in its own first line, exactly what should run it.

## Make it executable and run it

Saving the file is not enough. By default a new text file is not marked as a program,
so the shell will not run it directly. You add the **execute bit** with `chmod`:

```
chmod +x script.sh
```

That `+x` is the same execute [permission](/learn/linux-cli/permissions/) you have seen
on other programs — it tells the system this file is allowed to be run. Now you can run
it:

```
./script.sh
```

Note the **`./`** in front. The shell only looks for bare command names in the
directories on your [PATH](/learn/linux-cli/environment-variables/), and the current
directory is normally **not** on PATH (a deliberate safety choice, so a stray file
called `ls` in a folder can't hijack the real one). Writing `./script.sh` gives an
explicit location — "the file *here*, in this directory" — so the shell knows precisely
which file you mean.

If you would rather not set the execute bit, you can hand the file straight to bash:

```
bash script.sh
```

Here bash itself reads the file, so the shebang and the execute bit are not required.
It's a quick way to run a one-off script without changing its permissions.

## Comments and a first script

Any line beginning with `#` (after the shebang) is a **comment** — the shell ignores it.
Comments are notes to yourself and to whoever reads the script later, explaining *why* a
line is there. Use them freely.

Here is a complete first script. Save it as `hello.sh`:

```
#!/usr/bin/env bash
# hello.sh — my first script

echo "Hello from a script!"
echo "Today is:"
date
```

Make it runnable and run it:

```
chmod +x hello.sh
./hello.sh
```

The shell runs each line in order: it prints the greeting, prints a label, then runs
`date` and shows the current time. Nothing here is new — it is exactly what you would
get by typing those four commands by hand. The win is that now it is **one command**.

## Why script at all

The moment you notice you are typing the **same sequence of commands more than once**,
that sequence wants to be a script. Saving it buys you two things:

- **Repeatability** — it runs the same way every time, with no missed steps and no
  typos. A checklist that runs itself.
- **Automation** — a long routine collapses into a single name you can run any time,
  share with someone else, or hand to the computer to run *for you* on a schedule (see
  [scheduling with cron](/learn/linux-cli/scheduling-with-cron/)).

A script is the bridge from *using* the shell to *building* with it. Everything that
follows — variables, loops, functions — is about making these files do more.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the shebang names the interpreter that should run the script." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the shebang (<code>#!/bin/bash</code>) line do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Adds a comment the shell ignores</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Tells the system which interpreter should run the script</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Makes the file executable so you can run it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **script** is a text file of commands the shell runs top to bottom, as if typed.
- The **shebang `#!`** on line one names the interpreter — `#!/bin/bash` or
  `#!/usr/bin/env bash`.
- **`chmod +x script.sh`** sets the execute bit so the file can be run.
- Run it with **`./script.sh`** (the `./` is needed because the current directory is
  not on PATH), or with **`bash script.sh`** to skip the execute bit.
- Lines starting with **`#`** are comments the shell ignores.
- Scripting turns anything you type more than once into **one repeatable command**.

Next up: [variables, arguments & input](/learn/linux-cli/variables-and-input/)
