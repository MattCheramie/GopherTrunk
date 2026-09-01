# GT-TR-01.01 — Trunked radio

Trunked radio is how hundreds of user groups share a handful of frequencies — by never owning any of them.

Here is the problem it solves. Under conventional radio, every group gets its own fixed frequency. Police dispatch is always on channel A. Fire is always on channel B. Public works is always on channel C. Simple — but wasteful, because most of the time, most of those channels are silent.

Radio traffic is bursty. Any one group transmits only a small fraction of the time, and groups rarely all key up at once. A city with thirty groups on thirty frequencies is paying for thirty channels and using maybe four of them in any given second.

Trunking flips the model. Take a small pool of channels — say five — and let a computer hand them out on demand. When someone keys up, their radio asks for a channel. A coordinator called the control channel picks a free one, announces the assignment, and every radio in that user's group retunes to it together. When the talker is done and a hang timer expires, the channel goes back into the pool for the next call.

No group owns a frequency anymore. What a group owns is an identity — a talkgroup — and the system finds a physical channel for each call, one call at a time.

The reason this works is pure statistics — the same queuing theory that lets a telephone exchange serve a whole city with far fewer lines than subscribers. Traffic engineers measure offered load in erlangs and size the pool so that the probability of a call finding every channel busy — the grade of service — stays low. If it ever happens anyway, the system queues the request or plays a busy tone.

The payoff is dramatic: a well-sized pool of five channels can comfortably serve dozens of talkgroups that would each have demanded a permanent frequency under the conventional model.

The idea comes in several shapes. FDMA systems like P25 Phase 1 give each call its own frequency. TDMA systems like DMR and TETRA split each frequency into timeslots, so one channel carries two or even four calls at once. And a single site can stand alone, or be linked with others into a wide-area network covering a whole state. We will meet all of these later in the course.

For a scanner, trunking changes everything. You cannot just park on one frequency, because the conversation you care about scatters across the pool, call by call. GopherTrunk deals with this the same way the radios do: it decodes the coordinator's announcements and follows each call to wherever it lands — which means one software-defined radio can reconstruct every conversation on the system at once.

So: trunked radio shares a small pool of channels among many groups. A computer assigns a free channel to each call and takes it back afterward. And what a group keeps is not a frequency but an identity. The full write-up is linked below.
