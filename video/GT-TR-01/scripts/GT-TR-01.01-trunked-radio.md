# GT-TR-01.01 — Trunked radio
slug: trunked-radio
also_slugs: [conventional-radio]

**[V: title "Trunked radio"]**

**[V: conv-vs-trunk intro]** [CLIP c1 start]
Trunked radio is how hundreds of user groups share a handful of frequencies — by never owning any of them.

Here is the problem it solves. Under conventional radio, every group gets its own fixed frequency. Police dispatch is always on channel A. Fire is always on channel B. Public works is always on channel C. Simple — but wasteful, because most of the time, most of those channels are silent.

**[V: conv-vs-trunk idle]**
Radio traffic is bursty. Any one group transmits only a small fraction of the time, and groups rarely all key up at once. A city with thirty groups on thirty frequencies is paying for thirty channels and using maybe four of them in any given second. [CLIP c1 end]

**[V: pool grant]** [CLIP c2 start]
Trunking flips the model. Take a small pool of channels — say five — and let a computer hand them out on demand. When someone keys up, their radio asks for a channel. A coordinator called the {control channel|control channel} picks a free one, announces the assignment, and every radio in that user's group retunes to it together. When the talker is done and a hang timer expires, the channel goes back into the pool for the next call.

**[V: pool release]**
No group owns a frequency anymore. What a group owns is an identity — a {talkgroup|talkgroup} — and the system finds a physical channel for each call, one call at a time. [CLIP c2 end]

**[V: defcard erlang "Grade of service|The probability a new call finds every channel busy — the number a channel pool is sized against."]**
The reason this works is pure statistics — the same queuing theory that lets a telephone exchange serve a whole city with far fewer lines than subscribers. Traffic engineers measure offered load in erlangs and size the pool so that the probability of a call finding every channel busy — the grade of service — stays low. If it ever happens anyway, the system queues the request or plays a busy tone.

**[V: pool stats]**
The payoff is dramatic: a well-sized pool of five channels can comfortably serve dozens of talkgroups that would each have demanded a permanent frequency under the conventional model.

**[V: defcard variants "One idea, many shapes|FDMA · TDMA · single-site · multisite"]**
The idea comes in several shapes. {FDMA|F D M A} systems like {P25|P twenty-five} Phase 1 give each call its own frequency. {TDMA|T D M A} systems like {DMR|D M R} and {TETRA|tetra} split each frequency into timeslots, so one channel carries two or even four calls at once. And a single site can stand alone, or be linked with others into a wide-area network covering a whole state. We will meet all of these later in the course.

**[V: gt-tiein]**
For a scanner, trunking changes everything. You cannot just park on one frequency, because the conversation you care about scatters across the pool, call by call. GopherTrunk deals with this the same way the radios do: it decodes the coordinator's announcements and follows each call to wherever it lands — which means one software-defined radio can reconstruct every conversation on the system at once.

**[V: recap "trunked-radio|① Many groups share a small channel pool ② A computer assigns a channel per call, then reclaims it ③ Groups keep an identity, not a frequency"]**
So: trunked radio shares a small pool of channels among many groups. A computer assigns a free channel to each call and takes it back afterward. And what a group keeps is not a frequency but an identity. The full write-up is linked below.
