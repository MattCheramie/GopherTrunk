# GT-TR-01.04 — Channel grant

A channel grant is the single most important message on a trunked system: the moment it is broadcast, a call is starting, and the message says exactly where.

Everything trunking promises — many groups, few channels — comes down to this one transaction. A user keys up. Their radio sends a call request on the control channel. The site controller picks a free voice channel from the pool, packs an answer into a short signalling block, and broadcasts it: talkgroup 101, go to channel 3.

Every radio affiliated to talkgroup 101 hears that grant and retunes together, in a fraction of a second. So does every monitor following the system. One message, and an entire group — plus everyone listening in — moves to the same place at the same instant.

Look inside the message and there are three essentials. First, the target: a talkgroup for a group call, or a source-and-destination pair for a private call. Second, the channel — usually not a raw frequency, but a channel number the radio maps to a frequency through the system's channel plan. And third, on TDMA systems, a timeslot, because two calls can share one frequency in time.

A grant is broadcast once — but a radio might miss it. It was out of range, powered off, or busy on another channel. So while the call runs, the control channel repeats a grant update announcing the call in progress. A late arrival reads the update and joins mid-call. That is late entry, and it is why trunked systems feel seamless even as radios come and go.

The flavors mirror the traffic: group voice grants for the everyday case, unit-to-unit grants for private calls between two radio IDs, grant updates for late entry, and data grants that assign a channel for a short packet transfer instead of voice.

One more wrinkle: a grant is a promise of a free channel, and sometimes there isn't one. During a busy incident the controller may queue the request instead, and the grant arrives seconds later when a channel frees up. Grade of service, made audible.

GopherTrunk's trunk-following engine is, at heart, a grant reader. Every grant on the control channel is parsed in real time; the channel number runs through the decoded channel plan to become a frequency; and a receiver is tasked at that frequency — and that timeslot — before the first syllable of audio arrives. Repeat for every grant, and one wideband capture reconstructs the whole system's traffic.

So: a channel grant is the control-channel message that assigns a call to a voice channel. It names the target, the channel, and on TDMA the slot — and its updates repeat mid-call so latecomers can join. Full write-up linked below.
