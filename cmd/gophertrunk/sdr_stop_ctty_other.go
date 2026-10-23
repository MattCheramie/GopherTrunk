//go:build !unix

package main

// hasControllingTerminal is true off Unix: a Windows console delivers
// Ctrl-C to the process, so there is no su-style gap to cover.
func hasControllingTerminal() bool { return true }
