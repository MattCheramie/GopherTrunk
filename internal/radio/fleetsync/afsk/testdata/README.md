# FleetSync real-air fixtures

Two channelized slices of the on-air captures @v2maldo posted for issue
#1184 (the verification gate for #437): a Kenwood radio sending its
FleetSync-I and FleetSync-II ANI, **Fleet 107 / Unit 1772**, on
462.5625 MHz.

| File | Source | Content |
| --- | --- | --- |
| `fleetsync1_fleet107_unit1772_48k.cs16` | SDR# baseband WAV, 2.048 MS/s, 32-bit float, carrier at +424.5 kHz | 1.3 s from t = 1.0 s: two FleetSync-I ANI bursts |
| `fleetsync2_fleet107_unit1772_48k.cs16` | SDR# baseband WAV, 2.048 MS/s, 32-bit float, carrier at +323 kHz | 1.3 s from t = 0.8 s: two FleetSync-II ANI bursts |

Both were produced by the production `ccdecoder.Downconverter` (tune to
the carrier, 48 kHz output), then peak-normalised to 0.9 and written as
headerless interleaved int16 I/Q (`cs16`, little-endian). Replay one with:

```
GT_FLEETSYNC_IQ=internal/radio/fleetsync/afsk/testdata/fleetsync2_fleet107_unit1772_48k.cs16 \
GT_FLEETSYNC_FORMAT=cs16 GT_FLEETSYNC_RATE=48000 \
  go test ./cmd/gophertrunk -run 'TestFleetSyncReplay$' -v
```

`realair_test.go` decodes both through `afsk.Receiver` and asserts the
ANI. The full captures (94 MB and 127 MB) are on the issue.
