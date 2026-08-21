---
slug: remote-administration
title: Remote administration
description: Managing a board you can't physically reach — SSH keys instead of passwords, tmux for surviving disconnects, reading logs remotely, safe upgrades, and the habits that avoid locking yourself out.
keywords: ssh keys, ssh-copy-id, disable password authentication, tmux, remote logs, journalctl, safe remote upgrade, scp, locked out ssh
level: intermediate
status: full
prereq:
  - first-boot-and-ssh
  - users-and-updates
---

# Remote administration

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An appliance is administered entirely over the network, so the connection deserves
engineering. **SSH keys** replace passwords — stronger, unphishable, and typed by
nobody; once they work, **disable password login**. **tmux** keeps long-running
work alive when your connection drops — the difference between "my laptop slept"
and "my upgrade died halfway." Logs are read remotely (`journalctl`,
`ssh host command`), files move with **scp/rsync**, and every risky change follows
the golden rule: **never break the door you came in through** — test the new
connection before closing the old one.
</div>

From here on, assume the board is behind the sofa, in the attic, or at a relative's
house: physically inconvenient forever. This lesson upgrades your SSH workflow from
"can log in" to "administer with confidence," including the discipline that keeps
you from locking yourself out.

## Why keys instead of passwords?

A password is a secret you type — guessable by bots hammering every SSH port on
earth, and phishable from you. An **SSH key pair** is different: a private key that
never leaves your laptop and a public key you place on the board; login becomes a
cryptographic proof, nothing secret crossing the wire. Setup is two commands from
your PC:

```bash
$ ssh-keygen -t ed25519          # once per laptop; accept defaults, set a passphrase
$ ssh-copy-id matt@scanner.local # appends your public key to the board
$ ssh matt@scanner.local         # now logs in without the account password
```

Once key login works from every machine you'll use, close the password door in
`/etc/ssh/sshd_config`:

```text
PasswordAuthentication no
```

then `sudo systemctl restart ssh` — **but only after** testing a key login from a
*second terminal you keep open* (the golden rule, formalised below). The
passphrase on your private key protects it if the laptop is stolen; an SSH agent
means you type it once per session. ([SSH &amp; remote](/learn/linux-cli/ssh-and-remote/)
covers the mechanics more deeply.)

## What does tmux save you from?

An SSH session dies with your network — Wi-Fi blip, laptop lid, train tunnel — and
everything running in it dies too. Mid-`apt full-upgrade`, that's a genuinely bad
day. **tmux** is a terminal multiplexer: your shell runs inside a session *on the
board*, and your SSH connection merely views it. Disconnect and the work continues;
reconnect and reattach:

```bash
$ tmux new -s upgrade        # start a named session
# ... run the long thing ...
# connection drops — no matter
$ ssh matt@scanner.local
$ tmux attach -t upgrade     # everything still there, still running
```

> Rule of thumb: any remote command that would hurt to have die halfway — an
> upgrade, a large copy, a long test — runs inside tmux. It's one habit that
> deletes a whole category of disaster.

## How do you look around without logging in?

Administration is mostly *reading* — and SSH runs single commands as easily as
shells, which composes with everything:

```bash
$ ssh scanner.local 'journalctl -u gophertrunk -e -n 50'   # last 50 daemon lines
$ ssh scanner.local 'vcgencmd measure_temp; uptime'        # quick health
$ scp scanner.local:/var/lib/gophertrunk/recordings/2026-08-20/*.wav ./   # fetch files
$ rsync -av scanner.local:/etc/gophertrunk/ ./config-backup/  # sync configs down
```

`journalctl -f` over SSH is your remote tail; `scp`/`rsync` move recordings and
configs. These one-liners are also the raw material of scripted health checks —
[Monitoring your board](/learn/embedded/monitoring-your-board/) builds on exactly
this. (GopherTrunk's own status lives in its web console, one more remote window
into the same box.)

## How do you change things without sawing off the branch?

The remote administrator's nightmare is self-lockout: an SSH config typo, a
firewall rule, a bad network change — and the fix requires the physical access you
don't have. The discipline that prevents it:

- **Never break the door you came in through.** Changing SSH or network config?
  Keep your current session open, make the change, and test a **new** connection
  from a second terminal. Only when the new door opens do you close the old one.
- **Prefer changes that undo themselves.** A reboot reverts anything not persisted
  — sometimes that's a feature: test a risky setting non-persistently first, make
  it permanent only after it proves harmless.
- **Lean on the recovery layers.** [Watchdogs](/learn/embedded/watchdogs-and-recovery/)
  and boot-enabled services mean a reboot is always a safe last resort —
  `sudo reboot` and wait two minutes is a legitimate remote fix.
- **Keep the fallback path in your pocket.** The noted IP address (if mDNS
  breaks), and — worst case — the knowledge that the SD card can be pulled and
  edited in any PC. Physical access is the ultimate recovery; the goal is never
  to *need* it.

Upgrades follow the same shape: `apt full-upgrade` inside tmux, reboot at a time
you can watch it come back, then confirm the service returned
(`systemctl status gophertrunk`). Boring on purpose.

<div class="knowledge-check" data-quiz data-correct-msg="Right — verify the new door opens before closing the old one; the open session is your safety line." markdown="0">
  <p class="knowledge-check__q">Quick check: you're about to disable SSH password login on a remote board. What's the safe sequence?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Disable it, reboot, then test whether your key works</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Keep your session open, make the change, and test a fresh key login from a second terminal before trusting it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Do it from the board's own keyboard, since remote changes to SSH are impossible</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SSH keys** (ssh-keygen, ssh-copy-id) replace guessable passwords; disable
  `PasswordAuthentication` once keys are proven everywhere.
- **tmux** detaches work from your connection — every long or risky remote
  command runs inside it.
- Administer by **reading remotely**: `ssh host command`, `journalctl` over SSH,
  `scp`/`rsync` for files.
- **Never break the door you came in through** — test new connections before
  closing old ones; prefer changes a reboot undoes.
- The recovery layers (watchdogs, boot-enabled services, the pullable SD card)
  make `reboot` a safe remote tool and physical access a never-needed last
  resort.

Next up: [Backups &amp; images](/learn/embedded/backups-and-images/).
