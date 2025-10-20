# Git Quality Standards Plugin

**Source**: `~/.claude/skills/foundations/git_quality_standards/`
**Version**: 1.0.0
**Category**: Workflow Automation

---

## Overview

This plugin provides git workflow enforcement, validation, and automation tools for the trading-data-adapter-py repository. It is part of the broader git_quality_standards skill applied across all Trading Ecosystem repositories.

**Key Features**:
- ✅ Pre-push validation hooks
- ✅ Branch naming convention enforcement
- ✅ PR documentation requirements
- ✅ GitHub Actions CI/CD workflows
- ✅ Automated PR creation with `gh` CLI

---

## Directory Structure

```
.claude/plugins/git_quality_standards/
├── README.md                       # This file
├── scripts/                        # Workflow automation scripts
│   ├── install-git-hooks.sh        # Basic hook installer
│   ├── install-git-hooks-enhanced.sh  # Enhanced installer with templates
│   ├── create-pr.sh                # Automated GitHub PR creation
│   ├── pre-push-hook.sh            # Pre-push validation logic
│   ├── validate-repository.sh      # Repository structure validation
│   └── README.md                   # Script documentation
├── templates/                      # Configuration templates
│   ├── pull_request_template.md    # GitHub PR template
│   └── .validation_exceptions      # Files to exclude from validation
└── workflows/                      # GitHub Actions workflows
    ├── pr-checks.yml               # PR validation on GitHub
    └── validation.yml              # Repository CI validation
```

---

## Installation

This plugin is already installed. The installation was performed by:

```bash
# From repository root:
./.claude/plugins/git_quality_standards/scripts/install-git-hooks-enhanced.sh -y -y -y
```

**What was installed**:
- ✅ Pre-push hook in `.git/hooks/pre-push`
- ✅ Symlinks in `.github/workflows/` → plugin workflows
- ✅ Symlink in `.github/pull_request_template.md` → plugin template

---

## Usage

### Pre-Push Validation

**Automatic**: Runs before every `git push`

Validates:
1. Not pushing from protected branches (main/master)
2. Branch name follows convention: `type/epic-XXX-9999-milestone-behavior`
3. PR documentation exists in `docs/prs/`
4. PR has required sections (Summary, Quality Assurance, etc.)
5. TODO.md was updated (when applicable)

**Skip validation** (emergency only):
```bash
git push --no-verify
```

### Creating a GitHub PR

**Manual PR creation** (traditional):
```bash
# 1. Push branch
git push -u origin feature/epic-TSE-0003-data-adapter-foundation

# 2. Create PR via web or gh CLI
gh pr create --fill
```

**Automated PR creation** (recommended):
```bash
# One command does everything:
./.claude/plugins/git_quality_standards/scripts/create-pr.sh

# This will:
# - Find PR documentation in docs/prs/
# - Extract title from first # heading
# - Use full doc as PR description
# - Push branch to remote
# - Create GitHub PR via gh CLI
# - Return PR URL
```

**Requirements for automated PR**:
- GitHub CLI (`gh`) installed: `brew install gh` or see https://cli.github.com/
- Authenticated: `gh auth login`
- PR documentation exists: `docs/prs/{branch-name}.md`

---

## Validation Rules

### Branch Naming Convention

**Format**: `type/epic-XXX-9999-milestone-behavior`

**Valid types**:
- `feature/` - New functionality
- `fix/` - Bug fixes
- `docs/` - Documentation only
- `style/` - Code formatting (no logic change)
- `refactor/` - Code refactoring (no behavior change)
- `test/` - Adding/updating tests
- `chore/` - Maintenance, dependencies, tooling
- `ci/` - CI/CD pipeline changes

**Examples**:
```bash
✅ feature/epic-TSE-0003-data-adapter-foundation
✅ fix/epic-TSE-0001-prometheus-metrics-bug
✅ chore/epic-TSE-0001-foundation-add-git-quality-standards
```

### PR Documentation Requirements

**Location**: `docs/prs/{branch-name}.md` or `docs/prs/{pr-prefix}-epic-XXX-9999-*.md`

**Required sections**:
- `## Summary` - What changed and why
- `## Quality Assurance` - Testing performed
- `## Security & Dependencies` - Security review
- `## Deployment` - Deployment requirements
- `## Breaking Changes` - Breaking changes (if any)

**Template**: `.claude/plugins/git_quality_standards/templates/pull_request_template.md`

---

## GitHub Actions Workflows

### 1. PR Checks (`pr-checks.yml`)

**Triggers**: On pull request to `main`

**Validates**:
- PR title follows conventional commits format
- Branch name follows epic convention
- PR has required labels (optional)

### 2. Repository Validation (`validation.yml`)

**Triggers**: On push to branches, on pull requests

**Validates**:
- Markdown syntax (markdownlint)
- Directory structure
- Required files exist
- Code quality checks

---

## Customization

### Exclude Files from Validation

Edit: `.claude/plugins/git_quality_standards/templates/.validation_exceptions`

```bash
# Test files with intentional errors
test-fixtures/*

# Generated code
**/generated/*

# Third-party code
vendor/*
```

### Customize GitHub Actions

Edit workflow files in `.claude/plugins/git_quality_standards/workflows/`:
- `pr-checks.yml` - PR validation rules
- `validation.yml` - Repository validation rules

**Note**: Changes take effect immediately via symlinks in `.github/workflows/`

---

## Why This Plugin Directory?

**Separation of Concerns**:
- ✅ Component code in `src/`, `tests/`
- ✅ Workflow tooling in `.claude/plugins/`
- ✅ Clear ownership: `.claude/` is tooling, not deliverable code

**Benefits**:
1. Easy to identify workflow vs component files
2. Can be `.gitignore`'d if workflow is external
3. Portable - copy entire plugin to other repos
4. Upgradeable - sync from global skills when needed

**Symlinks allow GitHub integration**:
- GitHub Actions expects `.github/workflows/*.yml`
- GitHub expects `.github/pull_request_template.md`
- Symlinks satisfy GitHub while keeping plugin self-contained

---

## Updating This Plugin

To sync with latest git_quality_standards skill:

```bash
# From repository root:
rsync -av --delete \
  ~/.claude/skills/foundations/git_quality_standards/scripts/ \
  .claude/plugins/git_quality_standards/scripts/

rsync -av --delete \
  ~/.claude/skills/foundations/git_quality_standards/templates/ \
  .claude/plugins/git_quality_standards/templates/

# Reinstall hooks
./.claude/plugins/git_quality_standards/scripts/install-git-hooks-enhanced.sh -y -y -y
```

---

## Troubleshooting

### Pre-push hook not running

**Check**:
```bash
ls -la .git/hooks/pre-push
# Should be executable and exist
```

**Fix**:
```bash
./.claude/plugins/git_quality_standards/scripts/install-git-hooks.sh
```

### GitHub Actions workflows not running

**Check symlinks**:
```bash
ls -la .github/workflows/
# Should show symlinks to .claude/plugins/git_quality_standards/workflows/
```

**Fix**:
```bash
cd .github/workflows
ln -sf ../../.claude/plugins/git_quality_standards/workflows/pr-checks.yml .
ln -sf ../../.claude/plugins/git_quality_standards/workflows/validation.yml .
```

### PR creation fails

**Common causes**:
1. `gh` CLI not installed
2. Not authenticated: `gh auth login`
3. No PR documentation in `docs/prs/`
4. PR doc doesn't match branch name

**Debug**:
```bash
# Check gh CLI
gh --version

# Check authentication
gh auth status

# Check PR docs
ls docs/prs/
```

---

## Related Documentation

- **Global Skill**: `~/.claude/skills/foundations/git_quality_standards/SKILL.md`
- **Workflow Checklist**: `~/.claude/skills/foundations/git_workflow_checklist/SKILL.md`
- **PR Creation Guide**: `~/.claude/skills/foundations/create_pull_request/SKILL.md`
- **Project Plan**: `../../../project-plan/CLAUDE.md`

---

## Support

This plugin is part of the Trading Ecosystem standardization initiative (TSE-0001).

For issues or questions:
1. Check this README
2. Review global skill: `~/.claude/skills/foundations/git_quality_standards/SKILL.md`
3. Consult project plan: `../../../project-plan/CLAUDE.md`

---

**Last Updated**: 2025-10-20
**Applied To**: trading-data-adapter-py (first implementation)
**Status**: ✅ Active
