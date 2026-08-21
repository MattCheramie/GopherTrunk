---
layout: learn-hub
learn_module: apis
permalink: /learn/apis/
title: Learn APIs & Protocols — from newbie to expert
description: A free, structured module on APIs and protocols — what an API contract is, how HTTP and REST work, real-time push with WebSockets, server-sent events, and webhooks, binary RPC with gRPC and Protocol Buffers, designing and operating APIs, and a full case study of the REST, event, and streaming APIs of GopherTrunk's own daemon.
keywords: learn APIs, what is an API, REST tutorial, HTTP basics, WebSockets, server-sent events, webhooks, gRPC, protocol buffers, API design, API security, rate limiting, GopherTrunk API, streaming API
---

Every program you use is quietly talking to other programs — your browser to a web
server, your phone's weather app to a forecast service, GopherTrunk's web console to
the scanner daemon behind it. The rules of those conversations are **APIs** and
**protocols**: agreements about who may ask what, in which format, and what the answer
will look like. This module teaches you to read those agreements, use them from the
command line and from code, and eventually design your own — from a first raw HTTP
request all the way to binary gRPC streams.

**Who this is for.** Anyone who has wondered what "the API" actually is — hobbyists who
want to script their scanner, web developers filling in the protocol layer beneath
`fetch()`, and Go programmers heading toward backend work. It assumes you can run
commands in a terminal; the [Networking]({{ '/learn/networking/' | relative_url }})
module (how bytes cross the network) and the
[Go programming]({{ '/learn/programming-go/' | relative_url }}) module (the language of
the code samples) are ideal companions but not prerequisites.

**How the path works.** Six units climb from ideas to a working client. The first is
**foundations** — what APIs, protocols, and contracts are. The second is the workhorse:
**HTTP and REST**, dissected one request at a time. The third covers **real-time
push** — WebSockets, server-sent events, webhooks, and the backpressure problem behind
all of them. The fourth turns to **RPC and binary protocols** with gRPC and Protocol
Buffers. The fifth is the craft of **designing and operating** an API people can trust.
The last unit studies a working example — the REST, event, and streaming interfaces of
[GopherTrunk]({{ '/' | relative_url }})'s own daemon — and ends with you building a
client against them. Mark lessons complete as you go — your progress is saved in your
browser. New here?
**[Start with lesson 1: What is an API?](/learn/apis/what-is-an-api/)**
