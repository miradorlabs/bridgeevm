# CLAUDE.md

Project-specific guidance for Claude. Keep this file lean — it loads into
every conversation. Codebase architecture lives in the code; this file is
for conventions and workflow.

## Project

`bridgeevm` is a Go library that identifies cross-chain bridge events in
EVM transaction logs. Build a `Detector` once per chain, hand any
`*types.Log` to `Detector.Detect`, and get back the bridge name, leg type
(source / destination), and a correlation ID linking the leg to its
counterpart on the other chain.

- Module path: `github.com/miradorlabs/bridgeevm` (matches repo URL).
- Single Go module at the repo root. No `cmd/` — pure library.
- Go 1.26.1 (see `go.mod`).
- License: MIT.
- Status: pre-1.0. Breaking changes are allowed in `0.x.0` minor bumps;
  `0.x.y` patch releases never break callers.

Bridge configs are embedded JSON under `config/<chain>/*.json` and loaded
once via `sync.OnceValues`. `chains.go` declares an exported `Chain*`
string constant per `config/<chain>/` directory; a drift test in
`chains_test.go` pins the two sets together.

## Commit conventions

**Conventional Commits are required** — release-please parses commit
messages on `main` to compute the next SemVer bump and to regenerate
`CHANGELOG.md`. The `commitlint` workflow validates PR commits.

Allowed types and how they appear in releases:

| Type        | Section                  | Bump (pre-1.0) |
|-------------|--------------------------|----------------|
| `feat`      | Features                 | minor          |
| `fix`       | Bug Fixes                | patch          |
| `perf`      | Performance Improvements | patch          |
| `refactor`  | Code Refactoring         | patch          |
| `deps`      | Dependencies             | patch          |
| `docs`      | Documentation            | patch          |
| `ci`        | (hidden)                 | none           |
| `build`     | (hidden)                 | none           |
| `test`      | (hidden)                 | none           |
| `chore`     | (hidden)                 | none           |
| `style`     | (hidden)                 | none           |
| `revert`    | (hidden)                 | none           |

Breaking changes use `type!:` (e.g. `refactor!: rename Detector.Identify
to Detector.Detect`) or a `BREAKING CHANGE:` footer. While we're pre-1.0,
release-please bumps these as **minor**, not major (configured via
`bump-minor-pre-major: true`).

### Commit-message body rules

release-please v17 uses the spec-strict `@conventional-commits/parser`,
which tokenizes the **entire** message (subject + body + footers). If the
parser errors anywhere, the **whole commit is silently skipped** — even
when the subject is a valid `feat:`. That means a single bad body can
prevent a release PR from being generated at all. Rules:

- **No nested parens anywhere in the message.** The grammar treats `(` as
  opening a scope and only allows a flat `)` to close it. `foo(bar(baz))`
  in a body line will fail with `unexpected token '(' ... valid tokens
  [)]`. Rewrite as `foo of bar of baz`, or split across lines.
- **No unmatched parens.** Same reason.
- **Don't start a body line with `<word>:` where `<word>` looks like a
  conventional-commit type** (`feat:`, `fix:`, etc.) — the parser may
  read it as a second header. Use a different phrasing or indent it.
- **Squash-merge subjects must themselves be conventional.** GitHub's
  default squash subject is the PR title — set the PR title to a clean
  `type: subject` before merging. The bullet-list body that GitHub
  generates from sub-commits is fine as long as the rules above hold.

When in doubt, `git commit` then run `npx @conventional-commits/parser`
on the message before pushing — easier than diagnosing a missing release
PR after the fact.

### Dependabot commits

For Dependabot bumps to trigger patch releases, `.github/dependabot.yml`
must use `commit-message.prefix: deps` for both `gomod` and
`github-actions` ecosystems. With any other prefix (`chore`, `ci`,
`build`) the commits are hidden in `release-please-config.json` and
produce no bump.

### DCO sign-off

Every commit must be signed off (`Signed-off-by: ...` trailer). Always
commit with `git commit -s`. The `dco` GitHub workflow blocks PRs without
trailers on every commit.

### Authorship

**Solo author only.** Do not add `Co-Authored-By:` trailers (including
Claude). The DCO sign-off is the only trailer.

## Pre-push checks

Install hooks once: `make setup-hooks` (copies `.githooks/pre-push` and
`.githooks/commit-msg` into `.git/hooks/`).

Before pushing, run `make verify` — runs `go fmt`, `go vet`,
`golangci-lint run`, and `go test ./...`. CI runs the same set on every
PR.

## Linter

`golangci-lint` is pinned to a v2 release in two places that **must stay
in sync**:

- `Makefile` → `GOLANGCI_LINT_VERSION` (used by `make tools`, installed
  via `go install` so the binary builds against the local Go toolchain)
- `.github/workflows/ci.yml` → `version:` on `golangci-lint-action@v8`
  (downloads the prebuilt release binary)

If CI fails with *"the Go language version (goX.Y) used to build
golangci-lint is lower than the targeted Go version"*, the prebuilt
binary at the pinned version was compiled with an older Go than `go.mod`
targets — bump to a newer v2 release in **both** files.

`.golangci.yml` uses the v2 schema (`version: "2"`, top-level
`formatters:`, `linters: settings:`). Do not regress to v1 syntax.

## Releases

Tagging and CHANGELOG generation are automated by **release-please**.
The action runs on every push to `main`:

1. Scans conventional-commit messages since the last tag.
2. Opens (or updates) a long-lived release PR — `chore: release vX.Y.Z`
   — that bumps `.release-please-manifest.json` and regenerates
   `CHANGELOG.md`.
3. Merging that PR tags `vX.Y.Z` and creates the GitHub Release.

**Do not hand-edit `CHANGELOG.md`** — release-please owns it. Write good
commit messages instead; they become the changelog.

The git tag is the source of truth for the module version. There is no
`version.go` constant.

The release PR is opened with `GITHUB_TOKEN`, so other workflows do not
auto-trigger on it. If CI needs to run on the release PR, close-and-reopen
it or push an empty commit.

## Testing

- `go test ./...` — unit tests.
- `go test -race ./...` — race detector. CI runs this.
- `go test -bench=. -benchmem -run=^$ ./...` — benchmarks. The miss path
  and lookup-key construction are zero-alloc; the hit path allocates only
  the correlation ID string.

`testdata/` holds real on-chain logs, source/destination pairs per
protocol. Validation tests confirm both legs of a bridge produce the same
correlation ID.

Config-loading tests (`config_test.go`) inject malformed configs via
`testing/fstest.MapFS` against the `loadBridgeConfigs(fs.FS)` signature —
no filesystem round-trip needed. Use the same pattern when adding
load-time validation tests.

## Adding a new chain

1. Create `config/<chain>/<bridge>.json` files (one per bridge).
2. Add a `Chain<Name>` constant to `chains.go`.
3. `TestChainConstantsMatchConfigDirs` will fail until both are in
   place — that's the intent.

## Adding a new bridge on an existing chain

Just drop a JSON file into `config/<chain>/`. Validation runs at
`New()` time (lazy, via `sync.OnceValues`) and rejects: invalid topic
hash, invalid `bridgeTopic.type`, unsupported correlation type, or a
duplicate `(address, topic)` collision on the same chain.
