#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "${ROOT_DIR}"

OUT_DIR=${OUT_DIR:-docs/assets/screenshots}
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${OUT_DIR}"

BIN="${TMP_DIR}/tmc"
go build -o "${BIN}" ./cmd/tmc

"${BIN}" --help > "${TMP_DIR}/help.txt"
"${BIN}" version > "${TMP_DIR}/version.txt"
DEMO_ROOT="/Users/you/projects/mission-control-demo"
"${BIN}" init --root "${DEMO_ROOT}" --output "${TMP_DIR}/project.yml" --layout agent-lab --name "Mission Control Demo" >/dev/null
"${BIN}" dry-run --file "${TMP_DIR}/project.yml" > "${TMP_DIR}/dry-run.txt"
"${BIN}" dry-run --file "${TMP_DIR}/project.yml" --json > "${TMP_DIR}/dry-run-json.txt"

export TMP_DIR
perl -pi -e 's#\Q$ENV{TMP_DIR}\E/project\.yml#project.yml#g; s#/Users/[^/]+/Library/Caches#/Users/you/Library/Caches#g' "${TMP_DIR}"/*.txt

node - "${TMP_DIR}" "${OUT_DIR}" <<'NODE'
const fs = require("fs");
const path = require("path");

const [tmpDir, outDir] = process.argv.slice(2);

const captures = [
  {
    source: "help.txt",
    target: "tmc-help.svg",
    title: "tmc --help",
    command: "$ tmc --help",
    maxLines: 22,
  },
  {
    source: "dry-run.txt",
    target: "tmc-dry-run.svg",
    title: "tmc dry-run",
    command: "$ tmc dry-run --file project.yml",
    maxLines: 34,
  },
  {
    source: "dry-run-json.txt",
    target: "tmc-json.svg",
    title: "tmc JSON output",
    command: "$ tmc dry-run --file project.yml --json",
    maxLines: 32,
  },
  {
    source: "version.txt",
    target: "tmc-version.svg",
    title: "tmc version",
    command: "$ tmc version",
    maxLines: 8,
  },
];

function escapeXml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function truncateLine(line, width) {
  if (line.length <= width) return line;
  return `${line.slice(0, width - 1)}…`;
}

for (const capture of captures) {
  const body = fs.readFileSync(path.join(tmpDir, capture.source), "utf8").trimEnd();
  const lines = [capture.command, "", ...body.split("\n")]
    .slice(0, capture.maxLines)
    .map((line) => truncateLine(line, 106));
  const lineHeight = 18;
  const width = 980;
  const height = 78 + lines.length * lineHeight;
  const text = lines
    .map((line, index) => {
      const color = index === 0 ? "#a7f3d0" : "#d6deeb";
      return `<text x="28" y="${66 + index * lineHeight}" fill="${color}">${escapeXml(line)}</text>`;
    })
    .join("\n");

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-label="${escapeXml(capture.title)} terminal screenshot">
  <rect width="100%" height="100%" rx="12" fill="#0b1020"/>
  <rect x="0" y="0" width="100%" height="38" rx="12" fill="#111827"/>
  <circle cx="24" cy="19" r="6" fill="#ff5f57"/>
  <circle cx="44" cy="19" r="6" fill="#ffbd2e"/>
  <circle cx="64" cy="19" r="6" fill="#28c840"/>
  <text x="490" y="24" text-anchor="middle" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" font-size="13">${escapeXml(capture.title)}</text>
  <g font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" font-size="14">
${text}
  </g>
</svg>
`;
  fs.writeFileSync(path.join(outDir, capture.target), svg);
}
NODE

printf "wrote screenshots to %s\n" "${OUT_DIR}"
