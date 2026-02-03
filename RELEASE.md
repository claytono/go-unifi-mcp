# Release Rules

Read this document when preparing a new release.

## 1. Review Changes Since Last Release

Identify the most recent release tag and review all commits since then:

```bash
gh release list --limit 1
git log --oneline <last-tag>..HEAD
```

For each merged PR, read the PR description and understand the user-facing
impact. Categorize each change for the changelog (see section 3).

## 2. Determine the Next Version

This project follows [Semantic Versioning](https://semver.org/):

- **Patch** (0.1.1 → 0.1.2): Bug fixes, dependency updates, internal refactoring
  with no behavior change
- **Minor** (0.1.2 → 0.2.0): New features, new tools, new configuration options,
  new capabilities
- **Major** (0.x → 1.0): Reserved for stable API declaration; breaking changes
  while pre-1.0 bump minor

Present the proposed version with rationale to the user for approval.

## 3. Write Changelog Entries

Add a new version section to `CHANGELOG.md` in
[Keep a Changelog](https://keepachangelog.com/) format, validated by
[kacl-cli](https://gitlab.com/schmieder.matthias/python-kacl) (runs as a
pre-commit hook).

Categories (use only those that apply):

- **Added** — new features or capabilities
- **Changed** — changes to existing functionality
- **Deprecated** — features marked for future removal
- **Removed** — features removed in this release
- **Fixed** — bug fixes
- **Security** — vulnerability fixes

For housekeeping (Renovate dependency bumps, CI pipeline changes, and other
non-user-facing work), use **Changed** and summarize rather than listing
individually (e.g. "Pin and update GitHub Actions dependencies").

Writing style:

- Start each entry with an active verb (Add, Fix, Update, Remove, etc.)
- Reference PR numbers: `Add widget support (#42)`
- **Be verbose and descriptive.** Each user-facing entry should explain what the
  feature does, why it matters, and include concrete examples (before/after,
  sample arguments, representative output) so a reader understands the change
  without reading the code.
- Housekeeping entries under Changed are the exception — keep them to one
  summary line per group.

Add the new version's comparison link at the bottom of `CHANGELOG.md`:

```markdown
[X.Y.Z]: https://github.com/claytono/go-unifi-mcp/releases/tag/vX.Y.Z
```

## 4. Audit the README

Before releasing, review `README.md` against the changes in this release:

- Verify all new features, configuration options, and environment variables are
  documented
- Verify examples are accurate and reflect current behavior
- Verify tool counts and other concrete numbers are still correct
- Flag any gaps to the user before proceeding with the release

## 5. Pre-Release Checklist

- [ ] Changelog entries are complete and accurate
- [ ] Version link added at bottom of `CHANGELOG.md`
- [ ] README is up to date with all changes in this release
- [ ] Open PR with release changes, merge to main

## 6. Tag the Release

After the PR merges, update main, verify the merge commit is correct, then tag:

```bash
git checkout main
git pull
git log --oneline -3  # verify the merge commit contains the release changes
git tag -a vX.Y.Z -m "vX.Y.Z: brief summary"
git push origin vX.Y.Z
```

## 7. Post-Release Verification

After pushing the tag, the release workflow runs automatically. Verify:

- [ ] CI passes: `gh run watch --exit-status`
- [ ] Release created: `gh release view vX.Y.Z`
- [ ] Docker image published:
      `gh api orgs/claytono/packages/container/go-unifi-mcp/versions --jq '.[0].metadata.container.tags'`
- [ ] Homebrew formula updated: check `claytono/homebrew-tap` for commit
- [ ] MCP Registry published (runs as part of release workflow)

## 8. If Something Goes Wrong

- **Bad release notes**: Edit via `gh release edit vX.Y.Z --notes-file <file>`
- **Broken binary/image**: Fix the issue on main, then create a patch release
  following this same process
- **Wrong tag**: Delete and retag only if the release workflow hasn't completed;
  otherwise cut a new patch release
