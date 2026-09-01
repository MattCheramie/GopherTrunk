# GT-TR-01.03 — Talkgroup
slug: talkgroup
also_slugs: [radio-id]

**[V: title "Talkgroup"]**

**[V: tghop intro]** [CLIP c1 start]
A talkgroup is a channel that does not exist — and it is the thing you actually listen to on a trunked system.

On a trunked system, "Police Dispatch" is not a frequency. It is a number — a talkgroup ID — naming a group of users. When a member keys up, the system borrows a physical channel for that one call. The next call from the same group may land somewhere completely different.

**[V: tghop follow]**
Watch what that means over a minute of traffic: call one lands on channel 3, call two on channel 7, call three on channel 2. The frequency changes every time. The talkgroup number never does. The group is the stable identity; the frequency is a detail the system handles underneath. [CLIP c1 end]

**[V: tgprog membership]**
Every radio is programmed with the talkgroups it can select and monitor. Key up on one, and your radio sends a request carrying that talkgroup ID. The control channel grants a voice channel and announces the talkgroup, so every radio affiliated to the group retunes together and hears the same call — wherever it landed.

**[V: defcard tgid "Talkgroup ID|P25: 16-bit, scoped to WACN + System ID — alias tables map IDs to names"]**
The IDs are numeric and system-specific. On {P25|P twenty-five} a talkgroup ID is a sixteen-bit value scoped to the system's identity; {DMR|D M R} numbers its own way. Hobbyist databases publish alias tables that map the raw numbers to human names — police districts, fire, EMS, public works — which is how a scanner display turns "talkgroup 101" into something readable.

**[V: rid pair]** [CLIP c2 start]
Alongside the talkgroup rides a second number: the radio ID of the individual transmitting radio. Together they answer the two questions a listener has about every call — which group is this, and which unit is talking. Group identity for the many; unit identity for the one.

**[V: defcard variants "Kinds of groups|dispatch · tactical · patched · announcement"]**
Groups come in flavors. Dispatch groups carry a desk's steady traffic; tactical groups are short-lived channels stood up for an incident. A dispatcher can patch several talkgroups together so they temporarily hear one another, and an announcement group reaches every subordinate group at once. [CLIP c2 end]

**[V: gt-tiein]**
This is why every serious trunking scanner — GopherTrunk included — is organized around talkgroups, not frequencies. You build allow lists and priority lists of groups; each active call shows its talkgroup ID, its alias, and the transmitting radio ID; and the activity log counts traffic per group. The system hops frequencies constantly. You never have to care.

**[V: recap "talkgroup|① A virtual channel: a numbered group of users ② Calls hop frequencies; the ID stays put ③ You follow talkgroups, not frequencies"]**
So: a talkgroup is a virtual channel — a number naming a group of users. Its calls hop across physical frequencies, but the ID stays put, which is why you follow talkgroups and let the system worry about frequencies. Full write-up linked below.
