---
slug: ssh-and-remote
title: SSH & working remotely
description: Log in to another machine over the network with SSH — the first-connection host-key prompt, why key-based auth beats passwords, copying files with scp and rsync, running remote commands, an SSH config, and administering a headless Raspberry Pi or server.
keywords: ssh, secure shell, ssh keys, ssh-keygen, ssh-copy-id, scp, rsync, remote login, headless, ssh config, tmux, raspberry pi
level: intermediate
status: full
prereq:
  - first-commands
faq:
  - q: "What is SSH used for?"
    a: "SSH (Secure Shell) gives you an encrypted command-line session on another computer over the network. You use it to administer a server, a headless Raspberry Pi, or any machine you can't sit in front of — you get a normal shell, exactly as if you were typing at that machine directly."
  - q: "Are SSH keys more secure than a password?"
    a: "Yes. An SSH key pair is far stronger than a password: the private half never leaves your machine, so there's nothing to guess, phish, or leak in a breach. Key-based login is also more convenient — once set up you log in silently, without typing anything. It's the same kind of key you may already use for Git."
  - q: "What is the difference between scp and rsync?"
    a: "Both copy files over SSH. scp is simplest for a one-off copy of a file or two. rsync is smarter for directories and repeated syncs: it transfers only the parts that changed, so updating a big folder is fast, and it can resume and mirror. Reach for scp for quick copies and rsync for syncing directories."
  - q: "Why do my remote programs stop when I close the SSH connection?"
    a: "A foreground program you start over SSH is tied to that session, so closing the connection kills it. For anything long-running, detach it from the session: run it as a systemd service, or start it inside a terminal multiplexer like tmux or screen that keeps running after you disconnect."
---

# SSH & working remotely

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SSH** (Secure Shell) gives you an encrypted shell on another machine over the
network — the normal way to run a server or a headless Raspberry Pi you never
plug a monitor into. Log in with `ssh user@host`, then use the remote shell just
like your own. Prefer **key-based auth** over passwords: it's both safer and
silent once set up. Move files with **scp** for quick copies and **rsync** for
syncing directories. Closing the connection kills your foreground programs, so
run long jobs as a service or inside tmux.
</div>

Everything you've learned about the [command line](/learn/linux-cli/first-commands/)
so far assumed you were typing at the machine in front of you. SSH lifts that
restriction: the same shell, the same commands, but on a computer across the room
or across the world. It's how nearly every server and single-board computer is
run.

## Logging in — ssh

To open a session on another machine, give SSH a username and a hostname (or IP
address):

```bash
$ ssh alice@192.168.1.50
```

The **first** time you connect to a machine, SSH shows its host key and asks you
to confirm:

```bash
The authenticity of host '192.168.1.50' can't be established.
ED25519 key fingerprint is SHA256:Xh8k...9pQ.
Are you sure you want to continue connecting (yes/no/[fingerprint])?
```

This is SSH proving the remote machine is who it claims to be. Type `yes` and the
key is remembered, so you won't be asked again unless it changes. After you
authenticate, you land in a **normal shell** on the remote machine — your prompt
changes to show its hostname, and every command you run now runs *there*.

## Keys beat passwords

You can log in with the remote account's password, but a far better way is an
**SSH key pair**. A key pair has two halves: a **private** key that never leaves
your machine, and a **public** key you install on the machines you want to reach.
Generate one with `ssh-keygen`:

```bash
$ ssh-keygen -t ed25519 -C "you@example.com"
```

Then copy the **public** half to the remote machine — `ssh-copy-id` does this for
you and appends it to the right file:

```bash
$ ssh-copy-id alice@192.168.1.50
```

From then on, `ssh alice@192.168.1.50` logs you in **silently**, with nothing to
type. Key-based login is both more secure — there's no password to guess, phish,
or leak — and more convenient. If you've already set up
[SSH keys for Git](/learn/git/authentication/), these are the *same* keys; you
can reuse them here.

## Copying files

Two tools move files over the same SSH connection. Use **scp** ("secure copy")
for a quick, one-off transfer — the syntax mirrors `cp`, with `user@host:` in
front of the remote path:

```bash
$ scp report.txt alice@192.168.1.50:/home/alice/       # local  -> remote
$ scp alice@192.168.1.50:/var/log/app.log ./           # remote -> local
```

Use **rsync** to sync whole directories efficiently. It transfers only the parts
that changed, so re-running it to update a large folder is fast:

```bash
$ rsync -av ./captures/ alice@192.168.1.50:/home/alice/captures/
```

The trailing slashes matter to rsync — they mean "the *contents* of this
directory." Reach for scp when copying a file or two, and rsync when keeping
directories in step.

## Running commands & staying connected

You don't have to open an interactive session to run one command. Put the command
in quotes after the host and SSH runs it, prints the output, and exits:

```bash
$ ssh alice@192.168.1.50 "uptime"
 14:22:05 up 6 days,  3:41,  1 user,  load average: 0.08, 0.04, 0.01
```

Typing full `user@host` addresses gets old fast. A `~/.ssh/config` file lets you
define short aliases:

```
Host pi
    HostName 192.168.1.50
    User alice
```

Now `ssh pi` is enough. One catch: a program you start in the **foreground** over
SSH is tied to that session, so closing the connection **kills it**. For anything
long-running, detach it from the session — run it as a
[systemd service](/learn/linux-cli/services-and-systemd/), or start it inside a
terminal multiplexer like **tmux** or **screen**, which keeps
[the process](/learn/linux-cli/processes/) alive after you disconnect and lets you
reattach later.

## Managing a headless machine

A **headless** machine — a Raspberry Pi or server with no monitor, keyboard, or
mouse — is administered entirely over SSH. You flash its storage, put it on the
network, and from then on every configuration change, update, and log check
happens through an SSH session from your laptop. This is the everyday reality of
running a [single-board computer](/learn/intro-hardware/raspberry-pi-and-family/)
or a [home server](/learn/intro-hardware/home-servers/).

It's also exactly how you'd run
[GopherTrunk on a Linux box](/learn/linux-cli/running-gophertrunk-on-linux/): SSH
in, start it as a service so it survives your disconnect, and check on it remotely
whenever you like. No monitor ever gets plugged in.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the private key stays on your machine and there's nothing to phish." markdown="0">
  <p class="knowledge-check__q">Quick check: what's more secure than a password for SSH login?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A longer password</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">An SSH key pair</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Turning off the host-key prompt</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SSH** gives you an encrypted shell on another machine; `ssh user@host` logs
  you in, and the first connection asks you to confirm the host key.
- Once in, you have a **normal shell** on the remote machine — every command runs
  there.
- Prefer **key-based auth**: `ssh-keygen` to make a key, `ssh-copy-id` to install
  the public half. Safer and silent — the same keys you may use for Git.
- Copy files with **scp** for one-offs and **rsync** for efficient directory
  syncs.
- Closing the connection kills foreground programs — use a **service** or **tmux**
  for long-running work, and administer headless machines entirely over SSH.

Next up: [services & systemd](/learn/linux-cli/services-and-systemd/)
