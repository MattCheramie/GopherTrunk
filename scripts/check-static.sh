#!/bin/sh
# check-static.sh BINARY... — fail unless every BINARY is a statically linked
# ELF executable (no dynamic section, so no glibc loader or libdl needed).
#
# The Termux / Android builds (#1230) must be static: Android's Bionic libc has
# neither glibc's /lib/ld-linux*.so loader nor libdl.so.2, so a dynamically
# linked Linux binary does not start there. A default Linux build links libdl
# through purego's dlopen of libasound (internal/voice/player/alsa_linux.go);
# -tags nolibasound removes it. This catches the next purego/cgo import that
# would quietly make the Termux build dynamic again.
set -eu
command -v readelf >/dev/null || { echo "check-static: readelf not found (install binutils)" >&2; exit 2; }
status=0
for bin in "$@"; do
	if readelf -d "$bin" 2>/dev/null | grep -q '(NEEDED)'; then
		echo "check-static: $bin is dynamically linked:" >&2
		readelf -d "$bin" | grep '(NEEDED)\|interpreter' >&2 || true
		status=1
	elif ! readelf -h "$bin" >/dev/null 2>&1; then
		echo "check-static: $bin is not an ELF file" >&2
		status=1
	else
		echo "check-static: $bin is statically linked"
	fi
done
exit $status
