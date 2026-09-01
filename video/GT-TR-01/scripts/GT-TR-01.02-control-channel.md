# GT-TR-01.02 — Control channel
slug: control-channel
also_slugs: [voice-channel]

**[V: title "Control channel"]**

**[V: ccstream intro]** [CLIP c1 start]
The control channel is the one frequency on a trunked system that never carries a voice — and it is the only one worth decoding first.

Tune across a trunked site and you will find it immediately: a strong, continuous data carrier that never stops and never speaks. That is the control channel — the system's coordinator. It carries a nonstop stream of short signalling messages: registrations, call requests, and above all, channel grants.

**[V: ccstream messages]**
Every message answers a question some radio just asked. A radio powers on: the control channel registers it. A user keys up: the control channel grants the call and names the channel it will happen on. Because it announces where every call goes, locking onto this one frequency makes the entire system's activity visible. [CLIP c1 end]

**[V: ccpair roles]**
Its counterpart is the voice channel — or traffic channel. Control carries data and coordinates; a voice channel carries an actual conversation, and only for the brief life of a single call. A grant points a talkgroup at a voice channel, the audio flows, a short hang time keeps replies on the same channel, and then the frequency goes idle and rejoins the pool.

**[V: ccstream grantflow]** [CLIP c2 start]
The messages themselves are compact, fixed-format bursts, packed back to back on a fixed-rate downlink: {TSBKs|T S B Ks} on {P25|P twenty-five}, {CSBKs|C S B Ks} on {DMR|D M R}, and their equivalents elsewhere. Between grants, the stream also broadcasts the system's identity, a list of neighbor sites so radios know where to roam, and periodic status that keeps idle radios locked.

**[V: ccstream lateentry]**
And the announcements repeat. A radio that arrives mid-call — it was out of range, or just powered on — hears an update for the call in progress and joins late. That trick, late entry, only works because the control channel keeps narrating the system's state over and over. [CLIP c2 end]

**[V: defcard variants "Where the control lives|dedicated · rotating rest channel · composite"]**
Most systems dedicate one frequency to the job permanently — {P25|P twenty-five}, {DMR|D M R} Tier 3, {TETRA|tetra} — which is the easy case to find and monitor. Lighter {DMR|D M R} modes rotate the control function around the pool on a rest channel, and some systems interleave control signalling with voice on the same carrier.

**[V: gt-tiein]**
For GopherTrunk, the control channel is the map. Decode it, and every call on the system announces itself before it starts. GopherTrunk parses the grant, affiliation, and system messages in real time, tasks a receiver at each granted voice channel, and follows many conversations from one wideband capture. Finding an unknown system works the same way: hunt for the carrier that is always on and never talks.

**[V: recap "control-channel|① Data only — it never carries voice ② Announces grants, registrations, system info ③ Decode it first: it maps every call"]**
So: the control channel is the data-only coordinator of a trunked system. It announces every grant, registration, and system parameter. Decode it first — it is the map to everything else. Full write-up linked below.
