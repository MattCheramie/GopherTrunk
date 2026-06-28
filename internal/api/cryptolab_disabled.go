//go:build !cryptolab

package api

import "net/http"

// mountCryptolab is a no-op in the default build: the cryptolab research
// console is an opt-in subsystem, linked only with -tags cryptolab (see
// cryptolab_enabled.go). This keeps the cryptolab packages out of the default
// daemon binary.
func (s *Server) mountCryptolab(_ *http.ServeMux) {}
