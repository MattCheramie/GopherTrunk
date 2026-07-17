---
slug: viewing-and-editing
title: Viewing & editing files
description: How to read a file from the command line without opening an app — cat, less, head, and tail — plus a first, survivable look at editing in the terminal with the friendly nano and the ubiquitous vim.
keywords: cat, less, head, tail, nano, vim, view file linux, edit file terminal, tail -f, read file command line, terminal text editor
level: beginner
status: full
prereq:
  - navigating-and-listing
faq:
  - q: How do I view a file in the Linux terminal?
    a: "Use cat to dump a short file straight to the screen, or less to scroll through a long one page by page. For just the start or end of a file, use head or tail. None of these open an application — they print the text right in your terminal."
  - q: How do I exit less?
    a: "Press q. less takes over the screen to let you scroll, so the normal prompt is hidden until you quit — q returns you to it. Inside less you scroll with the arrows or space, and search with / followed by your text."
  - q: How do I quit vim?
    a: "Press Esc to make sure you're out of insert mode, then type :wq and Enter to save and quit, or :q! and Enter to quit without saving. vim is modal, so the Esc first is what gets you unstuck."
  - q: Which terminal editor should a beginner use?
    a: "nano — it shows its shortcuts along the bottom and behaves like a normal text box, so you can start typing immediately. Learn just enough vim to save and quit as well, because vim is the one editor you can count on finding on any server."
---

# Viewing & editing files

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
To **read** a file without opening an app, use **cat** (dump a small file),
**less** (scroll a big one), or **head** / **tail** (just the first or last
lines). To **edit**, **nano** is the friendly beginner's editor, while **vim**
is worth surviving because it's on every server. You've already learned to move
around and list files in
[navigating & listing](/learn/linux-cli/navigating-and-listing/) — now let's look
inside them.
</div>

Once you can move around the filesystem, the next thing you'll want is to *see
what's in a file* — and eventually to change it. Both happen right in the
terminal, no window, no mouse. This lesson covers a handful of reading commands
and gives you a first, survivable footing in two text editors.

## Reading files

**`cat`** prints a file straight to the screen. It's perfect for something
short — a config snippet, a note, a small script:

```
cat notes.txt
```

The whole file scrolls past. For a *long* file that's useless, because the top
flies off the screen. That's where **`less`** comes in:

```
less /var/log/syslog
```

`less` takes over the terminal and lets you move through the file at your own
pace: **arrows** or **space** to scroll, **`/`** followed by text to search,
**`n`** to jump to the next match, and **`q`** to **quit** back to the prompt.
Nothing is changed — `less` is read-only, so it's safe on anything.

Often you only want a peek. **`head`** shows the first lines of a file and
**`tail`** shows the last:

```
head config.yml
tail config.yml
```

Both default to ten lines; **`-n N`** sets how many. `tail -n 50 app.log` shows
the last fifty lines — handy because the *end* of a log is usually the part you
care about.

The star of the tail family is **`tail -f`**, which *follows* a file: it prints
new lines as they're written, so you can watch a log grow live while a program
runs. Press **Ctrl-C** to stop watching. You'll lean on this constantly when
[monitoring services and reading logs](/learn/linux-cli/monitoring-and-logs/).

## Editing with nano

Reading is safe; editing changes the file. The friendliest editor to start with
is **nano**. Open a file (it's created if it doesn't exist yet):

```
nano todo.txt
```

You just **type normally** — nano behaves like a plain text box, with your
cursor moved by the arrow keys. The clever part is the menu along the bottom,
which lists the shortcuts. The **`^`** symbol there means **Ctrl**, so:

- **`^O`** (Ctrl-O) — write **O**ut, i.e. save. Press Enter to confirm the name.
- **`^X`** (Ctrl-X) — exit. If you have unsaved changes it asks first.

That's genuinely all you need. nano is the editor to reach for whenever comfort
matters more than speed.

## Surviving vim

Sooner or later you'll land on a server where the habit is **vim** (or its older
sibling `vi`). It's powerful and fast once learned, but it trips up newcomers
because it's **modal** — the same keys do different things depending on the mode
you're in, and by default your typing doesn't go into the file. You don't need
to master it today. You need to not get *stuck*:

- Press **`i`** to enter **insert** mode — now typing edits the text as you'd expect.
- Press **`Esc`** to leave insert mode and go back to command mode.
- From command mode, type **`:wq`** and Enter to **save and quit**.
- Or type **`:q!`** and Enter to **quit without saving** (throw away changes).

Memorise those four and vim can never trap you. If you're ever lost, tap `Esc`
a couple of times first — that always returns you to command mode, where `:q!`
is your escape hatch.

## Which to use

Both editors edit **plain text**, and plain text is what almost all Linux
configuration is — no hidden formatting, just characters in a file. So the
choice is about comfort, not capability:

- Reach for **nano** when it's available and you want a gentle, obvious editor.
- Fall back to **vim** when it's the only thing installed — which, on a bare
  server, it often is.

Learning just enough of both means no machine can leave you unable to make a
quick edit.

## Creating files without an editor

You don't always need an editor at all. The shell can send a command's output
straight into a file with the **`>`** redirection operator:

```
echo "hello" > greeting.txt
```

`echo` prints its text, and `>` points that text into `greeting.txt`, creating
the file (or **overwriting** it if it already exists). This is just a taste —
redirection is a whole toolkit that gets the full treatment in
[pipes & redirection](/learn/linux-cli/pipes-and-redirection/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — q quits less and drops you back at the prompt." markdown="0">
  <p class="knowledge-check__q">Quick check: you opened a file in less. How do you exit back to the prompt?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Press Ctrl-X</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Press q</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Type :wq</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`cat`** dumps a short file; **`less`** scrolls a long one (arrows/space, `/` to search, **`q`** to quit).
- **`head`** and **`tail`** show the first or last lines; **`-n N`** sets how many.
- **`tail -f`** follows a file live — the way you watch a log grow.
- **nano** is the friendly editor: type normally, **`^O`** to save, **`^X`** to exit.
- **vim** is everywhere: **`i`** to insert, **`Esc`** to leave, **`:wq`** to save-and-quit, **`:q!`** to bail out.
- **`echo "text" > file`** creates a file with no editor at all.

Next up: [finding files & text](/learn/linux-cli/finding-things/)
