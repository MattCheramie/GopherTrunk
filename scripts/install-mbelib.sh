#!/usr/bin/env bash
# install-mbelib.sh — clone, build, and system-install the szechyjs
# mbelib library so `make build TAGS=mbelib` (and `make test
# TAGS=mbelib`) can link the libmbe.so / mbelib.h that the
# internal/voice/mbelib CGO wrapper requires.
#
# mbelib isn't packaged in standard distros, so the documented
# install path in docs/vocoders.md is "build from source". This
# script wraps that procedure so dev + CI environments can opt in
# with a single command:
#
#     scripts/install-mbelib.sh
#     make build TAGS=mbelib
#
# Override the install prefix or skip sudo by setting:
#
#     PREFIX=$HOME/.local USE_SUDO=0 scripts/install-mbelib.sh
#
# Patent + licensing context lives in docs/vocoders.md. AMBE+2 (the
# DMR / NXDN / P25 Phase 2 vocoder) is patent-encumbered; building
# this library is at the operator's discretion. The default
# GopherTrunk binary doesn't link mbelib at all.
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/szechyjs/mbelib.git}"
REPO_REF="${REPO_REF:-master}"
PREFIX="${PREFIX:-/usr/local}"
BUILD_DIR="${BUILD_DIR:-$(mktemp -d -t mbelib-build-XXXXXX)}"
USE_SUDO="${USE_SUDO:-1}"
# CMAKE_EXTRA_ARGS is passed straight through to the cmake invocation.
# Set this to e.g. "-DBUILD_TESTING=OFF" in CI to skip the gtest /
# gmock subbuilds that mbelib's tree pulls in for its unit tests but
# which add ~30 s of build time and add no value to a runtime install.
CMAKE_EXTRA_ARGS="${CMAKE_EXTRA_ARGS:-}"

log() { printf '==> %s\n' "$*" >&2; }

# Pick a sudo helper based on USE_SUDO + whether we're already root.
if [[ "$USE_SUDO" == "1" && "$(id -u)" != "0" ]]; then
  SUDO=sudo
else
  SUDO=
fi

# Required tools — fail fast with a clear message rather than a
# cryptic mid-build error.
for tool in git cmake make cc; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    log "missing required tool: $tool"
    log "install with your distro's package manager (e.g. apt-get install build-essential cmake git)"
    exit 1
  fi
done

log "cloning $REPO_URL ($REPO_REF) into $BUILD_DIR"
git clone --depth 1 --branch "$REPO_REF" "$REPO_URL" "$BUILD_DIR/mbelib"

log "configuring (prefix=$PREFIX, extra=$CMAKE_EXTRA_ARGS)"
mkdir -p "$BUILD_DIR/mbelib/build"
cd "$BUILD_DIR/mbelib/build"
# shellcheck disable=SC2086 # CMAKE_EXTRA_ARGS intentionally word-split
cmake -DCMAKE_INSTALL_PREFIX="$PREFIX" $CMAKE_EXTRA_ARGS ..

log "building"
make -j"$(nproc 2>/dev/null || echo 2)"

log "installing to $PREFIX (sudo=${USE_SUDO})"
$SUDO make install
$SUDO ldconfig 2>/dev/null || true

# Verify the install — header + shared object + pkg-config file.
if [[ ! -f "$PREFIX/include/mbelib.h" ]]; then
  log "FAIL: $PREFIX/include/mbelib.h missing after install"
  exit 2
fi
if ! ls "$PREFIX/lib"/libmbe* >/dev/null 2>&1; then
  log "FAIL: $PREFIX/lib/libmbe* missing after install"
  exit 2
fi

log "installed:"
ls -1 "$PREFIX/include/mbelib.h" "$PREFIX/lib"/libmbe* "$PREFIX/lib/pkgconfig/libmbe.pc" 2>/dev/null

log "done — build with: make build TAGS=mbelib"
