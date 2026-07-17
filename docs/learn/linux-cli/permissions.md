---
slug: permissions
title: File permissions
description: How Linux file permissions work — reading the rwx string in ls -l, what read, write and execute mean for files versus directories, and changing them with chmod (symbolic and octal) and ownership with chown and chgrp.
keywords: file permissions, chmod, chown, chgrp, rwx, octal, 755, 644, 600, ls -l, user group other, execute bit
level: intermediate
status: full
prereq:
  - users-and-groups
faq:
  - q: What does chmod 755 mean?
    a: "755 is octal shorthand for read-write-execute for the owner and read-execute for group and other. Each digit is one class of user, and its value adds up read (4), write (2) and execute (1): 7 = 4+2+1, 5 = 4+0+1. It is the usual mode for directories and for scripts everyone should be able to run."
  - q: What is the difference between chmod and chown?
    a: "chmod changes the permission bits — who may read, write or execute a file. chown changes ownership — which user and group the file belongs to, which in turn decides which permission class each person falls into. You often set ownership with chown first, then tune access with chmod."
  - q: How do I read the rwx string from ls -l?
    a: "The first character is the file type (- for a file, d for a directory). The next nine are three groups of rwx — for the owner, the group, then everyone else. A dash means that permission is absent, so -rw-r--r-- is a regular file the owner can read and write and everyone else can only read."
  - q: Why is chmod 777 a bad idea?
    a: "777 grants read, write and execute to everyone on the system, including write access to other users and services. It is a lazy way to make a permission error go away, but it removes the protection permissions exist to provide. Grant the narrowest access that works instead — usually 644 for files and 755 for directories."
---

# File permissions

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every file on Linux grants three permissions — **read**, **write**, **execute**
(**rwx**) — to three classes of people: the **user** (owner), the **group**, and
**other** (everyone else). You read those permissions with `ls -l`, change them
with **chmod**, and change who owns the file with **chown**. Reading and adjusting
that grid is core Linux, and it decides whether a script runs, a secret stays
private, or a whole folder is even reachable. New to owners and groups? Start with
[users & groups](/learn/linux-cli/users-and-groups/).
</div>

Permissions are how a multi-user system keeps people out of each other's files
and stops a stray program from touching what it shouldn't. The model is small —
three permissions times three classes of people — but it shows up everywhere, so
it is worth reading fluently rather than guessing.

## Reading ls -l

Run `ls -l` and every file gets a line that begins with a ten-character string:

```
-rw-r--r--  1 matt  staff  1420 Jul 17 09:31 notes.txt
```

That first block, `-rw-r--r--`, is the whole permission story. Read it in pieces:

- **Character 1** is the **file type**: `-` for a regular file, `d` for a
  directory, `l` for a symbolic link.
- **Characters 2–4** are the **user** (owner) permissions: `rw-`.
- **Characters 5–7** are the **group** permissions: `r--`.
- **Characters 8–10** are the **other** (everyone else) permissions: `r--`.

So `-rw-r--r--` is a regular file whose owner can **read and write** it, while the
group and everyone else can only **read** it. A dash in any slot means that
permission is switched off.

## What r, w, x mean

For an ordinary file the three letters are straightforward:

- **r (read)** — view the file's contents.
- **w (write)** — change or overwrite the contents.
- **x (execute)** — run the file as a program.

That last one matters for scripts. A shell script is just a text file until it has
the **execute bit**; without `x` the shell refuses to run it directly, no matter
how correct the code inside is. Getting a script to run for the first time is
exactly this problem — see
[your first shell script](/learn/linux-cli/first-shell-script/).

## …and for directories

The same three letters mean something different on a **directory**, and this is
the part that surprises people:

- **r (read)** — **list** the names of the entries inside.
- **w (write)** — **create, rename, or delete** entries inside it.
- **x (execute)** — **enter** the directory and reach the files within (traverse
  it, e.g. to `cd` in or open a file by path).

The twist is `x`: without it you cannot enter a folder even if you can read its
listing, and a folder with `x` but no `r` lets you open a file *if you already
know its name* but not list what is there. Because deleting a file is a change to
its **directory**, whether you can remove a file depends on the folder's `w`, not
the file's own permissions.

## Changing permissions — chmod

**chmod** ("change mode") sets the permission bits, and it takes two styles.

**Symbolic** style names a class (`u`ser, `g`roup, `o`ther, `a`ll), an operator
(`+` add, `-` remove, `=` set exactly), and the letters:

```
chmod u+x script.sh      # give the owner execute
chmod go-w notes.txt     # remove write from group and other
chmod a+r report.txt     # let everyone read it
```

**Octal** style sets all three classes at once with a three-digit number. Each
digit is the sum of **read = 4**, **write = 2**, **execute = 1**:

| rwx | binary | value | grants        |
|-----|--------|-------|---------------|
| `---` | 000 | **0** | nothing       |
| `r--` | 100 | **4** | read          |
| `r-x` | 101 | **5** | read, execute |
| `rw-` | 110 | **6** | read, write   |
| `rwx` | 111 | **7** | read, write, execute |

Put three of those digits together — one for user, group, other — and you have a
mode:

```
chmod 755 script.sh      # rwxr-xr-x  owner all, others read+run
chmod 644 notes.txt      # rw-r--r--  owner read/write, others read
chmod 600 secret.key     # rw-------  owner only, nobody else
```

## Changing ownership — chown & chgrp

Permissions decide what each **class** can do; ownership decides **who is in which
class**. **chown** ("change owner") sets the user (and optionally the group), and
**chgrp** sets just the group:

```
chown matt notes.txt          # matt is now the owner
chown matt:staff notes.txt    # owner matt, group staff
chgrp staff notes.txt         # change only the group
```

Handing a file to another user normally needs administrator rights, so these are
usually run with **sudo** — covered next in
[sudo & becoming root safely](/learn/linux-cli/sudo-and-root/).

## Common patterns & pitfalls

A few modes cover almost everything:

- **644** (`rw-r--r--`) — the default for ordinary files: you edit, others read.
- **755** (`rwxr-xr-x`) — directories, and scripts or programs everyone should run.
- **600** (`rw-------`) — secrets like private keys and password files; owner only.

The habit to avoid is **`chmod 777`**. It hands read, write, *and* execute to every
account and service on the machine, which makes a permission error vanish by
removing the protection entirely — including letting anyone overwrite the file.
When something won't open, work out the *narrowest* grant that fixes it (often just
adding `x` to a directory, or `644` to a file) rather than opening it to the world.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the last rwx triple is 'other', and r-- is read only." markdown="0">
  <p class="knowledge-check__q">Quick check: a file's permissions read <code>-rwxr-xr--</code>. What can "other" do with it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Read and execute it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Read only</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing at all</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Every file grants **rwx** to three classes: **user**, **group**, **other**.
- `ls -l` shows a ten-character string: file type plus three `rwx` triples.
- On **files** rwx means read / modify / run; a script needs the **execute** bit.
- On **directories** rwx means list / create-delete / enter — the `x` bit catches
  people out.
- **chmod** changes permissions (symbolic like `u+x`, or octal where r=4, w=2, x=1).
- **chown** and **chgrp** change ownership; use **644**, **755**, **600** as your
  defaults and avoid reflexive **777**.

Next up: [sudo & becoming root safely](/learn/linux-cli/sudo-and-root/)
