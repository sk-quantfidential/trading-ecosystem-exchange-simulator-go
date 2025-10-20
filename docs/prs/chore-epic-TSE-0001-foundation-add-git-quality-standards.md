# chore(epic-TSE-0001/foundation): add git quality standards infrastructure

## Summary

Added comprehensive git quality standards infrastructure including pre-push hooks, validation scripts, markdown linting, and GitHub Actions workflows.

**What Changed**:
- Added `.claude/plugins/git_quality_standards/` plugin architecture
- Added 7-check pre-push hook (protected branch, naming, PR docs, content, TODO updates, markdown, validation)
- Added repository validation script (`scripts/validate-all.sh` with 6 checks including branch-specific PR documentation)
- Added markdown linting configuration (`.markdownlint.json`)
- Added GitHub Actions workflows (pr-checks.yml, validation.yml)
- Added required documentation files (CONTRIBUTING.md, README.md, TODO.md)

## Motivation

Establish consistent git workflow quality standards across all repositories in the trading ecosystem to ensure:
- Proper epic tracking and milestone coordination
- Complete PR documentation before merging (matched to current branch)
- Consistent code quality and formatting
- Automated validation in CI/CD

## What Changed

### Validation Script CHECK 3 Fix
**IMPORTANT**: Fixed validate-all.sh CHECK 3 to verify PR documentation exists **for the current branch**, not just any PR doc in docs/prs/.

Now validates that PR documentation matches one of:
- `docs/prs/{prefix}-epic-XXX-9999-*.md` (e.g., `chore-epic-TSE-0001-foundation-*.md`)
- `docs/prs/{branch-name-with-dashes}.md` (e.g., `chore-epic-TSE-0001-foundation-add-git-quality-standards.md`)

This ensures developers create PR documentation specific to their feature branch before pushing.

### Plugin Architecture
```
.claude/plugins/git_quality_standards/
├── README.md
├── scripts/
│   ├── pre-push-hook.sh          # 7-check validation hook
│   ├── install-git-hooks-enhanced.sh
│   ├── create-pr.sh
│   ├── validate-repository.sh
│   └── README.md
├── templates/
│   ├── pull_request_template.md
│   ├── .validation_exceptions.template
│   └── .markdownlint.json.template
└── workflows/
    ├── pr-checks.yml
    └── validation.yml
```

### Pre-Push Hook (7 Checks)
1. ✅ Protected branch check
2. ✅ Branch naming convention
3. ✅ PR documentation required (branch-specific)
4. ✅ PR content validation
5. ✅ TODO.md updates verified
6. ✅ Markdown linting
7. ✅ Full repository validation

### Repository Validation (6 Checks)
1. ✅ Required files
2. ✅ Git quality standards plugin structure
3. ✅ **PR documentation for current branch** (validates branch-specific docs)
4. ✅ GitHub Actions workflows
5. ✅ Documentation structure
6. ✅ Markdown linting

## Testing

```bash
# Test validation on current branch
./scripts/validate-all.sh

# Should find: docs/prs/chore-epic-TSE-0001-foundation-add-git-quality-standards.md
# Or match: docs/prs/chore-epic-TSE-0001-foundation-*.md
```

## Impact

**Key Fix**: Validation now properly enforces branch-specific PR documentation, preventing false positives where any PR doc would pass validation.

## Related

- **Epic**: TSE-0001 Foundation Services & Infrastructure
- **Milestone**: Infrastructure Standardization
- **SKILL Fixed**: git_quality_standards
- **Script Fixed**: validate-all.sh CHECK 3

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
