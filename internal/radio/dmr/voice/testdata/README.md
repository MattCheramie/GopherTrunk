# DMR voice test fixtures

- `ep_issue1187_ptt1.json` — the PI header and the first three voice
  superframes (on-air AMBE+2 frames, 9 bytes each, MSB-first hex) of one
  DMR "Enhanced Privacy" (DMRA RC4) transmission from the issue #1187
  reporter's known-key capture (`DMR_EP_Test_KeyID_11_4E77AD0B51.wav`,
  96 kHz discriminator audio, simplex, colour code 2, TG 582743, key ID 11).
  The key is the one the reporter published on the issue for that test
  radio. Credit: mrscanner2008. Used by `TestEPCaptureIssue1187` to pin
  that the embedded IV names the *next* superframe's MI and that the
  descramble yields speech.
