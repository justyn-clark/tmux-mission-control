#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "${ROOT_DIR}"

VERSION=${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
DIST_DIR=${DIST_DIR:-dist}

targets=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

for target in "${targets[@]}"; do
  os=${target%/*}
  arch=${target#*/}
  package="tmc_${VERSION}_${os}_${arch}"
  outdir="${DIST_DIR}/${package}"

  mkdir -p "${outdir}"
  env CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o "${outdir}/tmc" \
    ./cmd/tmc

  cp README.md "${outdir}/README.md"
  tar -C "${outdir}" -czf "${DIST_DIR}/${package}.tar.gz" .
done

(
  cd "${DIST_DIR}"
  shasum -a 256 ./*.tar.gz > SHA256SUMS
)

printf "built release artifacts in %s for %s\n" "${DIST_DIR}" "${VERSION}"
