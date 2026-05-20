// Package voice decodes the DMR voice path: it recognises voice
// superframes in a dibit stream and extracts the AMBE+2 frames they
// carry.
//
// DMR voice is organised into 6-burst superframes (bursts A–F, 360 ms
// of audio). Burst A is framed by a voice sync word (BS/MS/DM Voice);
// bursts B–F replace the sync word with embedded signalling, so they
// carry no sync of their own and must be located by the fixed
// 132-dibit TDMA cadence relative to burst A. Each burst carries 216
// voice bits — three 72-bit on-air AMBE+2 frames — for 18 frames per
// superframe.
//
// This package produces the *on-air* AMBE frames (72 bits each, FEC
// still applied). Decoding the AMBE forward-error-correction down to
// the 49-bit vocoder payload, the ARC4 descramble for encrypted
// traffic, and wiring the decoder into the per-call composer/recorder
// all layer on top — see issue #276.
package voice
