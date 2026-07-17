---
slug: packet-capture-basics
title: Packet capture basics
description: When higher-level checks aren't enough, capture the actual packets on the wire — tcpdump from the shell and Wireshark in the GUI. How to capture on an interface, filter by host and port, read the summary lines, and tell traffic apart, plus the ethics of capturing responsibly.
keywords: packet capture, tcpdump, Wireshark, sniffing, pcap, network debugging, capture filter, protocol analysis, network traffic, read the wire
level: advanced
status: full
prereq:
  - tcp-and-udp
faq:
  - q: "What is packet capture?"
    a: "Packet capture is recording the raw network traffic that passes an interface, so you can inspect exactly what was sent and received. Tools like tcpdump and Wireshark read frames off the wire and show you the addresses, ports, protocols, and payloads — the ground truth when higher-level checks can't explain a problem."
  - q: "What is the difference between tcpdump and Wireshark?"
    a: "They capture the same traffic in different ways. tcpdump is a command-line tool: fast, scriptable, and ideal on a remote or headless machine over SSH. Wireshark is a graphical analyzer that decodes hundreds of protocols and lets you follow a stream visually. A common workflow is to capture with tcpdump to a file, then open that file in Wireshark to read it."
  - q: "How do I capture packets on a specific port?"
    a: "Give tcpdump a capture filter naming the interface and the port, for example tcpdump -i eth0 port 443. Only traffic to or from that port is recorded, which keeps the output readable and the capture file small. You can combine terms with host, src, dst, and boolean operators to narrow it further."
  - q: "Is running a packet capture legal?"
    a: "Capturing your own traffic on a network you own or are explicitly authorized to inspect is a normal, legitimate debugging task. Capturing other people's traffic without permission is a different matter and can be illegal. Always have authorization, and treat any capture as sensitive because it can contain private data."
---

# Packet capture basics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When pings and requests can't tell you *why* something is broken, stop guessing
and **read the wire**. **tcpdump** captures raw packets from the shell;
**Wireshark** does the same with a graphical decoder. Between them you can see
exactly what's flowing — in which direction, over which
[protocol](/learn/networking/tcp-and-udp/), and whether it's encrypted or
plaintext. Capture, filter to what matters, and interpret. Only ever capture
traffic you own or are authorized to inspect.
</div>

Every lesson so far has reasoned about traffic from the outside. Packet capture
lets you look *inside* the connection and watch the individual packets go by. It
is the most direct diagnostic tool on the network, and the last one you reach for
when nothing else explains the behaviour.

## When you need it

You've already checked the easy things. A [connectivity
test](/learn/networking/testing-connectivity/) says the host is reachable, and
[curl](/learn/networking/curl-and-http-tools/) gets *some* kind of response — but
the application still misbehaves and you can't see why. That's when you drop a
level and capture the packets themselves. It answers questions the higher-level
tools can only guess at:

- **Is any traffic flowing at all**, or is the connection silent?
- **In which direction** — are your packets leaving, and is anything coming back?
- **Is it encrypted or plaintext** — is [TLS](/learn/networking/tls-and-https/)
  actually in use, or is data going out in the clear?
- **Where does it stall** — which packet is the last one before things go quiet?

Packet capture answers all of these precisely, because it shows you the real
bytes rather than an interpretation of them.

## tcpdump — capture from the shell

**tcpdump** reads packets off a network interface and prints one summary line per
packet. You tell it which interface to listen on and a **capture filter** to limit
what it records — filtering at capture time keeps the output readable and the file
small.

```bash
# Capture HTTPS traffic to or from one host on eth0
$ sudo tcpdump -i eth0 -n host 192.168.1.50 and port 443

14:02:11.481203 IP 192.168.1.10.51544 > 192.168.1.50.443: Flags [S], seq 12…
14:02:11.482775 IP 192.168.1.50.443 > 192.168.1.10.51544: Flags [S.], seq 88…
14:02:11.482840 IP 192.168.1.10.51544 > 192.168.1.50.443: Flags [.], ack 1 …
```

You don't need to decode every field. Read the summary lines at a high level: the
source and destination `address.port` on each side, the direction of the `>`
arrow, and the [TCP](/learn/networking/tcp-and-udp/) flags — `[S]` a connection
opening, `[S.]` the reply, `[.]` an acknowledgement. That alone tells you whether a
handshake completed or died halfway. The `-n` flag skips name lookups so the
capture doesn't generate its own DNS traffic, and adding `-w capture.pcap` writes
the raw packets to a file instead of printing them.

## Wireshark — the GUI

**Wireshark** captures the same traffic with a graphical analyzer. It decodes
hundreds of protocols field by field, colour-codes packet types, and lets you
**follow a stream** to see a whole conversation reassembled in order rather than as
scattered packets.

It shines when the visual view helps: picking one connection out of a busy
capture, expanding a protocol's fields to see exactly what a server sent, or
scrubbing back and forth through a handshake. A common pattern is to capture on a
remote machine with `tcpdump -w capture.pcap` over
[SSH](/learn/networking/tcp-and-udp/), copy the file down, and open it in Wireshark
where you have a screen. Same packets, friendlier reading.

## What you can learn

A capture turns vague symptoms into specific facts:

- **Presence or absence of traffic.** If nothing appears, the packets aren't
  leaving your machine — look at routing, the firewall, or the address, not the
  far end.
- **Retransmissions.** The same segment sent again and again is a sign of
  [TCP](/learn/networking/tcp-and-udp/) trouble — loss or congestion on the path
  — not an application bug.
- **Encrypted vs. plaintext.** You can see whether
  [TLS](/learn/networking/tls-and-https/) is really negotiated, or whether data
  you assumed was protected is travelling in the clear.
- **Where a handshake stalls.** The last packet before the silence points at the
  exact step that failed — the SYN with no reply, the TLS hello that never
  completes.

## Ethics & permission

Packet capture is powerful precisely because it shows everything on the wire, and
that cuts both ways. Only ever capture on networks and traffic you **own or are
explicitly authorized to inspect** — your own machine, your own lab, a system you
have written permission to debug. Capturing other people's traffic without
authorization is a serious matter, not a technicality.

Treat every capture as sensitive. A `.pcap` can contain credentials, tokens, and
personal data in the clear, so store it carefully, share it narrowly, and delete
it when you're done. Handling captures responsibly is part of the same mindset
covered in [network security basics](/learn/networking/network-security-basics/).

## A familiar idea

If this feels familiar, it should. Decoding packets off the wire is the same
spirit as decoding signals off the air: **capture, filter, and interpret**. A
scanner tunes to a frequency, filters the band down to one channel, and turns raw
samples into meaning. A packet capture tunes to an interface, filters the flood
down to one host and port, and turns raw frames into meaning. GopherTrunk decodes
RF; here you decode network traffic — the discipline is identical.

<div class="knowledge-check" data-quiz data-correct-msg="Right — tcpdump captures raw packets from the command line." markdown="0">
  <p class="knowledge-check__q">Quick check: which command-line tool captures raw packets for inspection?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">ping</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">tcpdump</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">dig</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- When higher-level checks can't explain a problem, **capture the actual packets**
  to see exactly what's on the wire.
- **tcpdump** captures from the shell; a capture filter like `host X and port Y`
  keeps the output readable, and `-w` writes a `.pcap` file.
- **Wireshark** reads the same traffic in a GUI, decoding protocols and following
  streams — often on a file tcpdump captured elsewhere.
- A capture reveals **presence of traffic, direction, retransmissions, encryption,
  and where a handshake stalls**.
- Only capture traffic you **own or are authorized** to inspect, and handle the
  data responsibly — captures can hold private information.

Next up: Module 6 puts a service on the network — [running a server: binding & reachability](/learn/networking/running-a-server/)
