---
slug: users-and-groups
title: Users, groups & root
description: How Linux tracks who you are — usernames, numeric UIDs, and home directories — and how groups share access. Meet root, the all-powerful superuser, and see why ordinary-user limits are a feature, not a nuisance.
keywords: Linux users, groups, root, superuser, UID, whoami, id command, /etc/passwd, /etc/group, primary group, multi-user, who am I
level: beginner
status: full
prereq:
  - first-commands
faq:
  - q: "What is root on Linux?"
    a: Root is the superuser — the account with UID 0 that can do anything on the system, bypassing the normal file permissions ordinary users are bound by. Because a single mistake as root can wipe or break the whole machine, you normally work as an ordinary user and borrow root's powers only when you truly need them.
  - q: How do I see which user I am?
    a: Run whoami to print your username, or id for the fuller picture — your numeric UID, primary group GID, and every group you belong to. The commands who and w list everyone currently logged in to the machine.
  - q: "What is the difference between a user and a group?"
    a: A user is a single account with a name, a numeric UID, and a home directory. A group collects users together so they can share access to files or devices. Every user has one primary group and can belong to many additional groups at once.
  - q: Why can't I edit some files as a normal user?
    a: Linux is multi-user, so files are owned by accounts and protected by permissions. If a file belongs to another user or to root, your ordinary account is deliberately blocked from changing it. That limit stops mistakes and contains programs that misbehave — it is the foundation of file permissions.
---

# Users, groups & root

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Linux is **multi-user**: every account has a name and a number, and the system
tracks who you are so it can limit what you can touch. Accounts are collected into
**groups** to share access to files and devices. Above them all sits **root**, the
**superuser**, who can touch everything and is bound by no permission. Knowing which
account you are — and which you are *not* — is the first step in
[getting comfortable at the command line](/learn/linux-cli/first-commands/).
</div>

Coming from a single-user laptop, it is easy to assume the computer simply does what
you tell it. Linux is different: it was built for many people (and many programs) to
share one machine, so it always asks *who is this?* before it asks *what do you want?*
Once that clicks, permission errors stop being mysterious and start making sense.

## A multi-user system

Linux is a **multi-user** operating system. Every person or service that uses the
machine does so through an **account**, and each account carries three basic things:

- a **username** — the human-friendly name you log in with, like `matt` or `pi`;
- a numeric **UID** (user ID) — the number the kernel actually uses; the name is just
  a label on top of it;
- a **home directory** — usually `/home/username`, your own space for files and
  settings that other users can't disturb.

Because each account is separate, several people can share one computer safely. Your
files stay yours, their files stay theirs, and background **services** (the programs
that run the system) get their own locked-down accounts too. The machine keeps
everyone in their own lane.

## Groups

Working only account-by-account gets clumsy the moment two people need to share the
same files. That is what **groups** are for: a group collects users together so they
can be granted access as a unit, rather than one at a time.

Every user has one **primary group** — assigned when the account is created and used
for files that user makes — and can belong to **many additional groups** as well.
Groups often gate access to hardware: on many systems you must be in the `dialout`
group to use a serial device, or `plugdev` to reach USB gear like an SDR. Add a user
to the right group and the door opens; leave them out and it stays shut.

Groups have numeric **GIDs** just as users have UIDs, and the same name-on-a-number
idea applies.

## root — the superuser

One account stands apart from all the rest: **root**, the **superuser**. Root has
**UID 0**, and that number is special — the kernel lets root **do anything**. Root can
read, change, or delete any file regardless of who owns it, install software, manage
other accounts, and reconfigure the whole system. Root **bypasses permissions**
entirely; the limits that hold ordinary users simply don't apply.

That power is exactly why root is dangerous. There is no "are you sure?" backstop — a
mistyped command as root can erase files across the whole machine in an instant, and a
malicious program running as root owns everything. For that reason you normally do
**not** stay logged in as root for day-to-day work. Instead you live as an ordinary
user and step up to root's powers only for the specific task that needs them, which is
what [sudo and the root account](/learn/linux-cli/sudo-and-root/) is all about.

## Who am I?

A handful of commands tell you which account you are and who else is around:

- `whoami` — prints your current username. Short and to the point.
- `id` — the fuller answer: your **UID**, your primary **GID**, and every group you
  belong to. Run this when a permission is refused and you suspect a missing group.
- `who` and `w` — list the users currently logged in to the machine; `w` adds what
  each of them is doing.

Behind these commands sit two plain-text files. `/etc/passwd` is the list of accounts —
one line per user, holding the username, UID, primary group, and home directory (the
name is historical; passwords aren't stored there anymore). `/etc/group` is the
matching list of groups and who belongs to each. You can read both with the file
viewers from earlier lessons; they are simply the lists the system consults when it
decides who you are.

## Why limits are a feature

It is tempting to see "permission denied" as the computer getting in your way. It is
the opposite — it is the computer doing its job.

Ordinary-user limits do two valuable things. First, they **stop mistakes**: if your
account can't touch the system's core files, then a wrong command can only damage your
own stuff, not the whole machine. Second, they **contain compromised programs**: a web
browser or a decoder running under your account can only reach what your account can
reach, so a bug or an attack is boxed in rather than handed the keys.

This is the whole foundation of Linux **file permissions** — every file is owned by a
user and a group, and each carries rules about who may read, write, or run it. The
next lesson, [file permissions](/learn/linux-cli/permissions/), unpacks exactly how
those rules are written and read.

<div class="knowledge-check" data-quiz data-correct-msg="Right — root (UID 0) bypasses permissions and can override anything, which is exactly why you don't stay logged in as root." markdown="0">
  <p class="knowledge-check__q">Quick check: which user can override any file permission on the system?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Whoever owns the file</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">root (the superuser)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Any user in the same group</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Linux is **multi-user**: every account has a **username**, a numeric **UID**, and a
  **home directory**, so people and services share one machine safely.
- **Groups** collect users to share access to files and devices; a user has one
  **primary group** and can be in many.
- **root** is the **superuser** (UID 0) — it can do anything and **bypasses
  permissions**, which makes it both powerful and dangerous.
- You normally work as an ordinary user, not as root, and borrow root's powers only
  when needed.
- `whoami`, `id`, and `who`/`w` tell you who you are and who's logged in; `/etc/passwd`
  and `/etc/group` are the lists behind them.
- Ordinary-user limits are a **feature** — they stop mistakes and contain bad programs,
  which is the foundation of file permissions.

Next up: [file permissions](/learn/linux-cli/permissions/)
