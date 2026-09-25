//go:build integration

package phase2

// RotationLocks reports how many superframes this decoder has sliced under
// each of the four sync rotations since construction. Diagnostic for the
// rotated-sync detectors (issue #915): a stream with no residual-carrier
// false lock should show every count in slot 0.
func (d *SuperframeDecoder) RotationLocks() [4]int { return d.rotLocks }
