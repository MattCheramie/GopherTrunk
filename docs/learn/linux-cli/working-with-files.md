---
slug: working-with-files
title: Creating, copying & moving files
description: The everyday commands that build and rearrange your files and folders — mkdir and touch to create, cp to copy, mv to move and rename, rm and rmdir to delete — plus why rm has no undo.
keywords: mkdir, touch, cp, mv, rm, rmdir, delete file linux, copy files, move files, rename file linux, rm -rf danger, wildcards
level: beginner
status: full
prereq:
  - navigating-and-listing
faq:
  - q: How do I rename a file on Linux?
    a: "There is no dedicated rename command — you use mv. Run mv oldname newname and the file keeps its contents but takes the new name. Moving and renaming are the same operation: mv changes a file's path, and a path is just its location plus its name."
  - q: What is the difference between cp and mv?
    a: "cp copies: it makes a second, independent copy and leaves the original in place. mv moves: it relocates the one file (or renames it) so only one copy exists afterwards. Reach for cp when you want a backup or duplicate, and mv when you want to tidy or rename."
  - q: Can I undo an rm on Linux?
    a: "No. rm deletes immediately with no trash or recycle bin, and there is no built-in undo. That is why you check the path before pressing Enter, prefer rm -i to be prompted for each file, and treat rm -rf with real caution. If a file matters, back it up with cp first."
  - q: How do I delete a folder?
    a: "Use rmdir for an empty folder, or rm -r to remove a folder together with everything inside it. rm -r is recursive, so it deletes the directory and all its contents in one step — double-check the path first, because there is no undo."
---

# Creating, copying & moving files

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
These are the everyday commands that build and rearrange your files and folders.
**mkdir** makes a folder and **touch** makes an empty file; **cp** copies, **mv**
moves *and* renames (there is no separate rename command), and **rm** deletes.
The one to respect is **rm** — it has **no undo** and no trash can, so you check
the path before you press Enter. New to moving around the filesystem? Start with
[navigating & listing](/learn/linux-cli/navigating-and-listing/).
</div>

Once you can move around the filesystem, the next step is changing it: making
new folders, copying and moving things, and clearing away what you no longer
need. It is a small set of commands, each doing one predictable thing — and one
of them deletes with no safety net, so we will treat it with the care it
deserves.

## Making folders and files

To create a new folder, use **mkdir** ("make directory"):

```
mkdir projects
```

If you want a whole nested path at once, add **-p** ("parents"). It creates every
folder in the chain and does not complain if some already exist:

```
mkdir -p projects/gophertrunk/logs
```

Without -p, `mkdir projects/gophertrunk/logs` fails unless the parent folders
already exist. With -p, the whole tree appears in one command.

To create an empty file, use **touch**:

```
touch notes.txt
```

**touch** has a second job: if the file *already* exists, it does not erase it —
it just updates the file's timestamp to now. So `touch` either makes a new empty
file or "bumps" an existing one; it never overwrites your contents.

## Copying — cp

**cp** ("copy") makes a second, independent copy. The pattern is *source* then
*destination*:

```
cp notes.txt notes-backup.txt
```

That leaves `notes.txt` untouched and creates `notes-backup.txt` beside it. If
the destination is a **folder**, the copy keeps its name but lands there instead:

```
cp notes.txt projects/
```

To copy a whole folder — with everything inside it — you need **-r**
("recursive"), or cp will refuse:

```
cp -r projects projects-backup
```

So `cp source folder/` drops a copy *into* a folder, while `cp source newname`
makes a **renamed** copy in the current spot. Same command, different destination.

## Moving & renaming — mv

**mv** ("move") does two jobs with one command, because on Linux there is **no
separate rename command** — renaming *is* moving a file to a new name.

To **move** a file into another folder:

```
mv notes.txt projects/
```

To **rename** a file in place, "move" it to a new name:

```
mv notes.txt notes-final.txt
```

Both are the same operation under the hood: mv changes the file's path, and a
path is just its location plus its name. Unlike cp, mv does not need -r to move a
folder — `mv projects archive/` relocates the whole directory in one step.

## Deleting — rm (careful!)

**rm** ("remove") deletes files:

```
rm notes.txt
```

To delete a folder and everything in it, add **-r** (recursive):

```
rm -r projects
```

For an *empty* folder you can also use **rmdir**, which only succeeds if the
folder has nothing left inside — a small built-in safety check:

```
rmdir logs
```

Here is the rule that matters most: **rm has no undo.** There is no trash can, no
recycle bin, and nothing to restore from — the file is simply gone. Two habits
keep you safe. First, **read the path before you press Enter**. Second, use
**rm -i** ("interactive") when you want the shell to ask before each deletion:

```
rm -i notes.txt
```

The command to fear is **rm -rf** — recursive *and* forced, deleting a whole tree
without a single question. It is genuinely useful, but a wrong path (an accidental
space, the wrong folder) can erase far more than you meant. Always double-check
exactly what path you have typed before running it.

## Using wildcards

The shell can stand in for many filenames at once using **wildcards**. For a
quick taste, `*` means "any characters," so this copies every `.txt` file into a
`backup` folder in one go:

```
cp *.txt backup/
```

That saves you naming each file by hand — but wildcards deserve real care, since
they can just as easily feed a long list of files to **rm**. We give them the
full treatment in
[wildcards & expansion](/learn/linux-cli/wildcards-and-expansion/).

## Try it safely

Everything above is safe to practise in a throwaway folder. This sequence makes a
test directory, creates and copies a couple of files, renames one, then removes
the whole thing — nothing outside `sandbox` is touched:

```
mkdir sandbox
cd sandbox
touch one.txt two.txt
cp one.txt one-copy.txt
mv two.txt renamed.txt
ls
cd ..
rm -r sandbox
```

Run it line by line and watch what `ls` shows after each step — that is the
fastest way to build a feel for what each command does.

<div class="knowledge-check" data-quiz data-correct-msg="Right — there is no rename command, so mv oldname newname does the job." markdown="0">
  <p class="knowledge-check__q">Quick check: how do you rename a file on Linux?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">rename oldname newname</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">mv oldname newname</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">cp oldname newname</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **mkdir** makes folders (add **-p** for a nested path); **touch** makes an empty
  file or updates an existing file's timestamp.
- **cp** copies (**-r** for folders); copying into a folder keeps the name, copying
  to a new name makes a renamed duplicate.
- **mv** both moves and renames — there is no separate rename command.
- **rmdir** removes an empty folder; **rm -r** removes a folder and its contents.
- **rm has no undo** — no trash, no recovery. Check the path, use **rm -i** to be
  prompted, and treat **rm -rf** with real caution.

Next up: [viewing & editing files](/learn/linux-cli/viewing-and-editing/)
