---
slug: sudo-and-root
title: sudo & becoming root safely
description: What the root superuser is, why living as root is dangerous, and how sudo lets you borrow root's power for a single command — with your own password, a log entry, and the least-privilege habits that keep one typo from wrecking the system.
keywords: sudo, root, superuser, su, sudoers, admin, privilege, least privilege, wheel group, sudo -i, linux permissions
level: intermediate
status: full
prereq:
  - users-and-groups
faq:
  - q: What is the difference between sudo and su?
    a: "sudo runs a single command as root and then hands control straight back to you, asking for your own password and logging what you did. su switches you into another user's session — su - opens a full root shell you stay in until you exit. sudo is the safer default because the elevated power lasts exactly one command."
  - q: Why do I need sudo to install software or edit /etc?
    a: "Those actions change the whole system, not just your own files, so they are reserved for root. Installing packages, editing configuration under /etc, and starting or stopping services all touch shared parts of the machine. sudo lets your normal account perform one such action at a time without you having to log in as root."
  - q: Is it dangerous to run everything with sudo?
    a: "Yes — that defeats the point. Root has no safety rails, so every typo runs unrestricted and a wrong path in a delete command can wipe the system. Use sudo only for the specific commands that truly need it, read the command before you press Enter, and never pipe a downloaded script straight into a root shell without inspecting it first."
  - q: Who is allowed to use sudo?
    a: "Membership in the sudo group (called wheel on some distributions), or a matching rule in the sudoers file, grants the right. An administrator sets this up; a fresh account usually cannot use sudo until it is added. That gate is deliberate — the power to become root is handed out on purpose, not by default."
---

# sudo & becoming root safely

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**root** is the all-powerful superuser, and living as root is a mistake waiting to
happen — every typo runs unrestricted. **sudo** borrows root's power for a single
command, asks for *your* password, and logs it, which is far safer than logging in
as root. The habit that keeps you safe is **least privilege**: use ordinary rights
for ordinary work, reach for sudo only when a task truly needs it, and always
**think before you sudo**. Knowing who owns what starts with
[users & groups](/learn/linux-cli/users-and-groups/).
</div>

Most of the command line only touches your own files, and your normal account can do
all of it. But some jobs — installing software, editing system config, managing
services — change the whole machine, and Linux reserves those for **root**. This
lesson is about reaching that power *briefly and deliberately*, instead of sitting in
it all day.

## Why not just log in as root?

**root** (the "superuser") can do anything: delete any file, change any setting, stop
any process. There are no guard rails, because root *is* the guard. That sounds
convenient until you remember that computers do exactly what you type, mistakes
included.

Stay logged in as root and two things get dangerous at once. Every command you run —
including the careless ones — executes with unlimited power, so a single typo can wreck
the system. And every *program* you launch inherits that power too, so an ordinary tool
with a bug, or a script you didn't fully read, runs unrestricted.

The principle that fixes this is **least privilege**: do your work with the *smallest*
rights that get the job done, and take on more only for the moment you need it. Working
as a normal user limits the **blast radius** of any mistake to your own files. That is
why seasoned operators almost never log in as root.

## sudo — power for one command

**sudo** ("superuser do") is how you take on root's power for *just one command*. You
prefix the command with `sudo`, and only that command runs as root — the moment it
finishes, you are back to your ordinary self:

```
sudo apt install gophertrunk
```

Three things make this safe. sudo asks for **your own password**, not a shared root
password, so nobody has to know a master secret. It **logs** the action, so there is a
record of who ran what as root. And the elevated power lasts exactly one command
instead of a whole session.

Not every account may do this. The right comes from membership in the **sudo** group
(called **wheel** on some distributions) or a matching rule in the **sudoers** file,
which an administrator sets up. On a machine you installed yourself, your first account
usually has it already.

## root's shell

The prompt tells you where you stand. An ordinary user's prompt ends in `$`; a root
prompt ends in `#`. If you ever see that **`#`**, every command you type is running with
full power — treat the whole session with care.

You *can* open a root shell and stay there:

```
sudo -i        # start an interactive root shell
su -           # switch to root's own login shell (needs the root password)
```

Both drop you at a `#` prompt where every command runs as root until you `exit`. Use
them **sparingly** — for a long run of admin commands where typing `sudo` each time is
genuinely tedious — and leave the moment you are done. For one-off tasks, a plain
`sudo <command>` is almost always the better choice.

## The dangerous ones

A couple of patterns cause most of the real damage, and both are worth a deliberate
pause before you press Enter.

The first is a **recursive delete on the wrong path**. `rm -rf` removes a folder and
everything under it without asking, and with `sudo` in front there is nothing to stop
it — point it at the wrong directory and it is simply gone. Read the path twice.

The second is **piping a downloaded script straight into a root shell**:

```
curl https://example.com/install.sh | sudo bash
```

That hands an unread script from the internet full control of your machine. Instead,
download it, **inspect** it, and only then run it:

```
curl -O https://example.com/install.sh
less install.sh          # read what it actually does
sudo bash install.sh     # run it once you trust it
```

The habit to build is a healthy pause before Enter on anything with `sudo`: *do I know
exactly what this command touches?*

## Everyday uses

Used well, sudo is unremarkable — the small set of tasks that legitimately need root:

- **Installing and updating software** — package managers write into system-wide
  locations, so they need root. See
  [package management](/learn/linux-cli/package-management/).
- **Editing system configuration** — files under **`/etc`** belong to root because they
  shape the whole machine; changing them needs elevation.
- **Managing services** — starting, stopping, and enabling background services is an
  administrative act. See
  [services & systemd](/learn/linux-cli/services-and-systemd/).

In each case the pattern is the same: an ordinary account, borrowing root's power for
one specific, deliberate command.

<div class="knowledge-check" data-quiz data-correct-msg="Right — sudo runs that one command as root and hands control straight back, so you never sit in root's unlimited power." markdown="0">
  <p class="knowledge-check__q">Quick check: you need to run one command that requires admin rights. Safest approach?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Log in as root and work there for a while</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Prefix it with sudo — don't stay logged in as root</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Give your normal account root's password and share it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **root** is the superuser with unlimited power and no safety rails.
- Living as root is dangerous: every typo, and every program, runs unrestricted.
- **least privilege** — work as a normal user and elevate only when needed — limits the
  blast radius of any mistake.
- **sudo** runs one command as root, asks for *your* password, and logs it; the right
  comes from the sudo/wheel group or the sudoers file.
- A `#` prompt means root; `sudo -i` or `su -` open a root shell — use them sparingly and
  `exit` promptly.
- Pause before Enter on `sudo rm -rf` and never pipe an unread script into `sudo bash`.

Next up: [processes & jobs](/learn/linux-cli/processes/)
