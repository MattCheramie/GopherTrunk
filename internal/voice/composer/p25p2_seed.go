package composer

// p25p2SeedWaitMaxBursts bounds the bursts a Phase 2 voice chain holds while
// it waits for its descramble seed (issue #1364; see p25p2.SiteSeeds): 10 s
// of one carrier (a burst every 30 ms). A site repeats its Network Status
// Broadcast every few seconds, so this covers the wait; a call that outlasts
// it keeps its newest bursts.
const p25p2SeedWaitMaxBursts = 334
