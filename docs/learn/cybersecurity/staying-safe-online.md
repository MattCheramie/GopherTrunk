---
slug: staying-safe-online
title: Staying safe online
description: A plain-language checklist of everyday habits that stop most attacks — keeping software updated, using a password manager with MFA, backing up your data, spotting scams, encrypting your devices, using a VPN on untrusted Wi-Fi, and minding your digital footprint.
keywords: online safety, personal security, password manager, MFA, updates, backups, phishing, VPN, device encryption, screen lock, privacy, scams
level: beginner
status: full
prereq:
  - passwords-and-mfa
faq:
  - q: What's the single most effective thing I can do to stay safe online?
    a: "There isn't one magic setting, but the closest thing is turning on automatic updates everywhere and using a password manager with multi-factor authentication. Most real-world attacks reuse an already-fixed flaw or a leaked password, so patching promptly and never reusing a password shuts the door on the majority of them. Add regular backups and a healthy suspicion of urgent messages and you've covered the everyday threats."
  - q: Do I really need a VPN to be safe?
    a: "A VPN is useful, not essential. It encrypts your traffic between your device and the VPN server, which mainly helps on untrusted public Wi-Fi where someone nearby might snoop. On your own network with modern HTTPS websites, most traffic is already encrypted, so a VPN adds less than people assume. Treat it as one tool for specific situations, not a substitute for updates, strong passwords, and backups."
  - q: How do backups protect me from attacks?
    a: "Ransomware works by encrypting your files and demanding payment to unlock them. If you have a recent, tested backup — ideally with one copy kept offline or somewhere the attacker can't reach — you can simply restore your data instead of paying. The same backup saves you from a failed hard drive, a lost phone, or a mistaken deletion, so it protects against accidents and attacks alike."
  - q: How can I tell if a message is a scam?
    a: "Scams almost always push urgency — a threat, a deadline, or a too-good reward — to stop you thinking. Slow down and check the sender independently, hover over links before clicking, and never share a login, one-time code, or payment because a message told you to. When in doubt, contact the organisation through a number or address you already trust rather than the one in the message."
---

# Staying safe online

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
You don't need to be an expert to be hard to attack. Turn on **updates**
everywhere, use a **password manager + MFA**, keep **backups**, and carry a
**healthy suspicion** of anything urgent. A short list of personal habits blocks
the vast majority of everyday attacks — most of which are automated and simply
move on to an easier target. Start with
[passwords & MFA](/learn/cybersecurity/passwords-and-mfa/) and build from there.
</div>

Almost none of the attacks that hit ordinary people are clever. They reuse a flaw
that was fixed months ago, a password that leaked from another site, or a message
that panics you into clicking. That's good news: the same handful of habits
defeats nearly all of them, and none of them require deep technical skill.

## Keep everything updated

The most common way in is an already-known, already-patched flaw. Attackers scan
for devices running old software because the fix — and the exact weakness it
closes — is public. If you haven't installed the update, you're the easy target.

Turn on **automatic updates** for your operating system, your browser, your apps,
and your phone, and let them install promptly rather than clicking "remind me
later" for weeks. This one habit closes the door on a huge share of attacks
before they start. For the bigger picture of reducing what can be attacked in the
first place, see [hardening systems](/learn/cybersecurity/hardening-systems/).

## Password manager + MFA

Passwords fail in predictable ways: people reuse them, so one leak unlocks many
accounts, and people pick guessable ones. A **password manager** fixes both by
generating a long, unique, random password for every site and remembering them
for you — you only memorise the one that unlocks the manager.

Then add **multi-factor authentication (MFA)** to your important accounts (email,
bank, anything that can reset other accounts). MFA means a stolen password alone
isn't enough to log in, because the attacker still lacks your second factor. This
pairing is the biggest security win for the least effort you'll find anywhere;
the full walkthrough is in
[passwords & MFA](/learn/cybersecurity/passwords-and-mfa/).

## Back up your data

Backups don't stop an attack — they make one survivable. Ransomware encrypts your
files and demands payment; a hard drive dies without warning; a phone gets left in
a taxi. In every case a recent backup turns a disaster into an inconvenience.

Make backups **regular** (automatic if you can) and **test** that you can actually
restore from them — an untested backup is just a hope. Keep at least one copy
**offline** or somewhere an attacker who gets into your machine can't reach, so
ransomware can't encrypt your backup along with everything else. For how this fits
into defending your devices, see
[malware & endpoints](/learn/cybersecurity/malware-and-endpoints/).

## Recognise scams

Most successful attacks don't break the technology — they trick the person. The
tell is nearly always **urgency**: a threat, a countdown, a prize that vanishes if
you don't act now. That pressure exists to stop you thinking.

So slow down. **Verify before you click or pay**: check who really sent a message,
look at where a link actually goes, and if it claims to be your bank or a service
you use, contact them through a number or app you already trust — not the one in
the message. And never hand over a **one-time code, password, or payment** because
someone asked; no legitimate organisation needs your login code. There's a whole
lesson on how these tricks work in
[social engineering](/learn/cybersecurity/social-engineering/).

## Protect your devices and connections

If a device is lost or stolen, the question is whether the thief gets your data
too. **Enable device/disk encryption** (built into modern phones and laptops) and
set a **screen lock** with a PIN, passcode, or biometric. Together they turn a
lost device into a lump of scrambled data instead of an open filing cabinet.

On **untrusted public Wi-Fi** — cafés, airports, hotels — be cautious, and use a
**VPN** when appropriate to encrypt your traffic to a server you trust. A VPN
isn't a magic shield, and modern HTTPS websites are already encrypted, but it
genuinely helps when you can't trust the network around you. See
[VPNs](/learn/networking/vpns/) for what one does and doesn't protect.

## Mind your footprint

Every piece of information you put online, and every permission you grant an app,
is something that can later be lost, leaked, or turned against you. You can't
share nothing — but you can be deliberate.

Think before posting details that answer security questions (birthdays, pet
names, your street). Review the **permissions** apps ask for and deny the ones
they don't need — a flashlight app has no reason to read your contacts. Less
exposed data is simply less to lose. For a fuller treatment of controlling your
information, see
[privacy & data protection](/learn/cybersecurity/privacy-and-data-protection/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — updates, a manager with MFA, and backups cover the everyday threats." markdown="0">
  <p class="knowledge-check__q">Quick check: which set of habits gives the most protection for the least effort?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A brand-new laptop and an expensive antivirus subscription</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Auto-updates, a password manager with MFA, and regular backups</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Turning off your Wi-Fi whenever you're not using it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Most attacks are automated and reuse known flaws or leaked passwords — a few
  habits stop nearly all of them.
- **Keep everything updated** so you're not exposed by an already-fixed weakness.
- Use a **password manager + MFA** — the biggest win for the least effort.
- Keep **regular, tested backups**, ideally with an offline copy, to survive
  ransomware and hardware failure alike.
- **Slow down on urgency**, verify before you click or pay, and never share codes
  or credentials.
- **Encrypt and lock your devices**, be careful on public Wi-Fi, and mind what you
  share and which permissions apps get.

Next up: [wireless & RF security](/learn/cybersecurity/rf-and-radio-security/)
