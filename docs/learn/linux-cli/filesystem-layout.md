---
slug: filesystem-layout
title: The filesystem hierarchy
description: How Linux organises everything into a single tree that starts at the root directory / — the neighbourhoods you'll meet (/home, /etc, /var, /usr), the difference between absolute and relative paths, and how ~, . and .. name a location.
keywords: linux filesystem, directory structure, root directory, /etc, /home, /var, absolute path, relative path, FHS, home directory, everything is a file
level: beginner
status: full
prereq:
  - first-commands
faq:
  - q: What is the root directory in Linux?
    a: "The root directory is /, the single top of the filesystem tree. Every file and folder on the system lives somewhere below it, and external disks are mounted into the tree rather than given separate drive letters. Note that / (the root of the tree) is different from /root, which is the home directory of the administrator account."
  - q: What is the difference between an absolute and a relative path?
    a: "An absolute path starts at the root directory / and names a location in full, like /home/you/notes.txt, so it means the same thing from anywhere. A relative path starts from your current directory, like notes.txt or ../up, so what it points to depends on where you are."
  - q: What does the ~ symbol mean in Linux?
    a: "~ (tilde) is shorthand for your home directory, usually /home/yourname. So ~/notes.txt and /home/yourname/notes.txt point to the same file, and typing cd with no argument takes you home as well."
  - q: Where do configuration files live on Linux?
    a: "System-wide configuration conventionally lives in /etc. Logs and other data that changes as the system runs live in /var. Your own settings live in your home directory, often in hidden files or folders whose names start with a dot."
---

# The filesystem hierarchy

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Linux keeps everything in **one tree** that starts at the **root, `/`** — there
are no `C:` or `D:` drive letters. Learn the few neighbourhoods you'll actually
visit, then learn to name any spot two ways: an **absolute path** from `/`
(`/home/you/notes.txt`) or a **relative path** from where you're standing
(`notes.txt`). **`~`** is always shorthand for **your home** directory. Get these
and the terminal stops feeling like a maze. New to the shell? Start with
[your first commands](/learn/linux-cli/first-commands/).
</div>

When you came from Windows or a Mac's Finder, files felt scattered across drives
and folders with no obvious order. Linux is tidier than it first looks: there is
exactly one tree, every location has a name, and the layout follows a shared
convention. Once you can picture the tree, `cd` and `ls` become a walk through a
neighbourhood you know.

## One tree, no drive letters

On Windows each disk gets its own letter — `C:`, `D:`, a USB stick as `E:`. Linux
does it differently. There is a **single tree**, and its top is the **root
directory**, written as a plain forward slash: **`/`**. Everything — programs, your
documents, system settings, even hardware — hangs off `/` somewhere.

So where do extra disks go? They are **mounted** *into* the tree at an empty
folder, rather than becoming a separate drive. A USB stick might appear at
`/media/you/USB`, an extra hard disk at `/mnt/data`. To you it's just another
folder deeper in the same tree; there's no separate `E:` to switch to. One tree,
many places — that's the whole model.

## The folders that matter

The tree has many branches, but a beginner only meets a handful. Here are the
neighbourhoods worth knowing by name:

- **`/home`** — where **your** files live. Each user gets a folder here, e.g.
  `/home/matt`. This is where you'll spend almost all your time.
- **`/etc`** — **system-wide configuration**. Plain-text settings files for the
  machine and its services live here (the name is often read as "et cetera").
- **`/var`** — **variable** data that changes as the system runs: logs, mail
  queues, caches. When something misbehaves, `/var/log` is where you look — see
  [monitoring & logs](/learn/linux-cli/monitoring-and-logs/).
- **`/usr`** and **`/bin`** — the **programs**. Most installed software and the
  commands you type (like `ls` and `cd`) live under these.
- **`/tmp`** — **scratch** space for temporary files. Anything here may be wiped
  on reboot, so never keep something you care about in `/tmp`.
- **`/root`** — the **administrator's** home directory. Note this is *not* the same
  as `/`; it just happens to share the word "root".
- **`/dev`** — **devices**. Disks, terminals, and other hardware appear here as
  special files (more on that below).
- **`/opt`** and **`/mnt`** — **`/opt`** holds some optional add-on software;
  **`/mnt`** is a common spot to mount extra disks.

You don't need to memorise every branch. Recognising these on sight is enough to
navigate confidently. This layout is a shared convention — the **Filesystem
Hierarchy Standard (FHS)** — which is why most Linux systems put things in the
same places.

## Absolute vs. relative paths

A **path** is just the name of a location in the tree. There are two ways to write
one, and knowing which is which saves a lot of confusion.

An **absolute path** starts at the root, `/`, and spells out the full route:

```
/home/you/notes.txt
```

Because it begins at `/`, it means the same file no matter where you currently
are. Absolute paths are unambiguous.

A **relative path** starts from your **current directory** — wherever you happen to
be standing. If you're already in `/home/you`, then:

```
notes.txt        the file in this directory
./notes.txt      the same thing — . means "here"
../up            go up one level first, then into "up"
```

Two special names show up constantly:

- **`.`** means **this** directory (the one you're in).
- **`..`** means the **parent** directory, one level up toward the root.

And one very handy shorthand:

- **`~`** (tilde) means **your home directory**, usually `/home/yourname`. So
  `~/notes.txt` is the same file as `/home/yourname/notes.txt`, from anywhere.

A quick rule of thumb: if a path starts with `/` it's absolute; if it starts with
`~` it's home-relative; anything else is relative to where you are right now.

## Everything is a file

Linux takes a striking shortcut: **almost everything is presented as a file**. Not
just your documents — hardware and system information show up as files too. A disk
appears under **`/dev`** (for example `/dev/sda`); live details about the running
system appear under **`/proc`**. You won't touch these on day one, but it's worth
knowing the idea exists, so that reading from a "file" to talk to a device later
doesn't come as a surprise. It's part of what makes the command line so
composable: the same tools that read text files can often read a device too.

## Try it

Open a terminal and walk the tree. `pwd` (**p**rint **w**orking **d**irectory)
tells you where you are; `cd` (**c**hange **d**irectory) moves you:

```
$ pwd
/home/you
$ cd /etc            # jump somewhere by absolute path
$ pwd
/etc
$ cd ..              # go up one level (to /)
$ pwd
/
$ cd home/you        # relative path from here (no leading slash)
$ pwd
/home/you
$ cd ~               # ~ takes you straight home
$ pwd
/home/you
```

Notice the pattern: a leading `/` means "start from the root," while a name
without one means "start from here." `..` steps up, `~` warps home.

<div class="knowledge-check" data-quiz data-correct-msg="Right — /etc is the conventional home for system-wide configuration." markdown="0">
  <p class="knowledge-check__q">Quick check: where do system-wide configuration files conventionally live?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">/home</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">/etc</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">/tmp</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Linux has **one tree** starting at the **root `/`** — no drive letters; disks
  **mount** into the tree.
- Key neighbourhoods: **`/home`** (your files), **`/etc`** (config), **`/var`**
  (logs and changing data), **`/usr`** and **`/bin`** (programs), **`/tmp`**
  (scratch).
- An **absolute path** starts at `/` and means the same thing everywhere; a
  **relative path** starts from your current directory.
- **`~`** is your home; **`.`** is here; **`..`** is one level up.
- Almost **everything is a file**, including devices (`/dev`) and system info
  (`/proc`).

Next up: [navigating & listing files](/learn/linux-cli/navigating-and-listing/)
