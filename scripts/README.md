# Git Quality Standards - Portable Scripts

This directory contains portable, self-contained scripts for implementing git quality standards in any project.

## Quick Start

```bash
# 1. Install git hooks (one command, run from repository root)
bash scripts/install-git-hooks.sh

# 2. Run validation suite
bash scripts/validate-repository.sh
```

## Scripts Overview

### `install-git-hooks.sh`

**Purpose**: One-command installation of git hooks

**What it does**:
- Copies `pre-push-hook.sh` to `.git/hooks/pre-push`
- Makes hooks executable
- Verifies installation

**Usage**:
```bash
bash scripts/install-git-hooks.sh
```

**Requirements**:
- Must be run from repository root
- Requires `.git/` directory
- Expects `scripts/pre-push-hook.sh` to exist

**Customization**: None needed - works out of the box

---

### `pre-push-hook.sh`

**Purpose**: Comprehensive pre-push validation with 6 checks

**What it validates**:
1. Not pushing from protected branch (main/master)
2. Branch name follows convention: `type/epic-XXX-9999-milestone-behavior`
3. PR documentation exists in `docs/prs/`
4. PR file has required sections (Summary, Testing, Changes)
5. TODO.md was updated (if applicable)
6. Full validation suite passes (if `validate-repository.sh` exists)

**Branch Types**:
- `feature` → PR prefix `feat`
- `fix` → PR prefix `fix`
- `docs` → PR prefix `docs`
- `style` → PR prefix `style`
- `refactor` → PR prefix `refac`
- `test` → PR prefix `test`
- `chore` → PR prefix `chore`
- `ci` → PR prefix `ci`

**Customization Points**:

1. **Project Code**: Change the project code pattern
   ```bash
   # Line 51: Update regex for your project codes
   BRANCH_REGEX="^(feature|fix|docs|style|refactor|test|chore|ci)/epic-[A-Z]{3}-[0-9]{4}-.+"
   # Change {3} to {2,4} for 2-4 letter codes
   ```

2. **PR Documentation Location**: Change where PR docs are stored
   ```bash
   # Line 96: Update PR documentation path
   if [[ -d "docs/prs" ]]; then
   # Change to your location, e.g., "docs/pull-requests" or ".github/prs"
   ```

3. **TODO.md Check**: Disable if you don't use TODO.md
   ```bash
   # Lines 190-242: Comment out entire CHECK 5 section
   ```

4. **Validation Suite**: Point to your validation script
   ```bash
   # Line 253: Update validation script path
   if [[ ! -f "scripts/validate-all.sh" ]]; then
   # Change to your script, e.g., "scripts/validate.sh"
   ```

**Skip Hook Temporarily**:
```bash
git push --no-verify  # Skip all git hooks
```

---

### `validate-repository.sh`

**Purpose**: Portable validation suite for any project

**What it validates**:
1. Git repository status
2. Required documentation files (README.md, etc.)
3. Branch naming convention
4. Markdown syntax (if markdownlint installed)
5. PR documentation structure
6. Git hooks installation

**Exit Codes**:
- `0` - All validations passed (warnings OK)
- `1` - Validation errors found

**Usage**:
```bash
# Run all validations
bash scripts/validate-repository.sh

# In CI/CD
./scripts/validate-repository.sh || exit 1
```

**Customization Points**:

1. **Required Documentation**: Add/remove required files
   ```bash
   # Line 59: Update required documentation list
   required_docs=("README.md")
   # Add your files: required_docs=("README.md" "ARCHITECTURE.md" "API.md")
   ```

2. **Optional Documentation**: Add optional files
   ```bash
   # Line 60: Update optional documentation list
   optional_docs=("TODO.md" "CLAUDE.md" "CONTRIBUTING.md")
   ```

3. **Branch Convention**: Customize project code pattern
   ```bash
   # Line 88: Update branch name regex
   BRANCH_REGEX="^(feature|fix|docs|style|refactor|test|chore|ci)/epic-[A-Z]{2,4}-[0-9]{4}-.+"
   # Adjust {2,4} for your project code length
   ```

4. **Markdown Validation**: Add custom ignore patterns
   ```bash
   # Line 107: Add custom ignore patterns
   MARKDOWNLINT_CMD="$MARKDOWNLINT_CMD --ignore \"path/to/ignore\""
   ```

5. **Disable Specific Checks**: Comment out entire check sections
   ```bash
   # To disable CHECK 4 (Markdown validation):
   # Comment out lines 102-137
   ```

**Dependencies**:
- `markdownlint-cli` (optional): `npm install -g markdownlint-cli`
- No Python dependencies (unlike full validate-all.sh)

---

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Repository Validation

on:
  push:
    branches: [ main, 'feature/**' ]
  pull_request:
    branches: [ main ]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Setup Node.js for markdownlint
      uses: actions/setup-node@v4
      with:
        node-version: '18'

    - name: Install markdownlint
      run: npm install -g markdownlint-cli

    - name: Run validation suite
      run: bash scripts/validate-repository.sh
```

### GitLab CI Example

```yaml
validate:
  stage: test
  image: node:18
  before_script:
    - npm install -g markdownlint-cli
  script:
    - bash scripts/validate-repository.sh
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
```

---

## Validation Exceptions

All scripts support `.validation_exceptions` file for excluding files from validation:

```bash
# .validation_exceptions
# Exclude specific files
deprecated/old-file.md

# Exclude with glob patterns
archived/**/*.md
vendor/**/*.md
node_modules/**

# Template files with intentional issues
templates/EXAMPLE_TEMPLATE.md
```

**Format**:
- One pattern per line
- `#` for comments
- Glob patterns supported (`**` for recursive)
- Relative to repository root

---

## Troubleshooting

### Hook Not Running

**Problem**: Git push doesn't trigger pre-push hook

**Solutions**:
1. Verify installation: `ls -la .git/hooks/pre-push`
2. Check executable: `chmod +x .git/hooks/pre-push`
3. Reinstall: `bash scripts/install-git-hooks.sh`

### Validation Fails on Valid Branch

**Problem**: Branch name is correct but hook rejects it

**Solutions**:
1. Check project code length in regex (lines 51, 88)
2. Verify epic number format (4 digits)
3. Check branch type spelling (must be exact)

### Markdown Validation Not Running

**Problem**: Scripts skip markdown validation

**Solutions**:
1. Install markdownlint: `npm install -g markdownlint-cli`
2. Verify installation: `markdownlint --version`
3. Check PATH includes npm global bin directory

### PR Documentation Not Found

**Problem**: Hook can't find PR documentation

**Solutions**:
1. Create `docs/prs/` directory: `mkdir -p docs/prs`
2. Add PR file: `cp template.md docs/prs/feature-epic-XXX-9999-title.md`
3. Check filename matches pattern: `{prefix}-epic-XXX-9999-*.md`
4. Commit PR file before pushing

---

## File Permissions

All scripts should be executable:

```bash
chmod +x scripts/install-git-hooks.sh
chmod +x scripts/pre-push-hook.sh
chmod +x scripts/validate-repository.sh
```

This is automatically handled by `install-git-hooks.sh`.

---

## Migration from Other Systems

### From Existing Git Hooks

If you have existing git hooks:

1. **Backup existing hooks**:
   ```bash
   cp .git/hooks/pre-push .git/hooks/pre-push.backup
   ```

2. **Merge validation logic**:
   - Copy checks from old hook into `pre-push-hook.sh`
   - Or call old hook from new hook:
     ```bash
     # In pre-push-hook.sh, before exit 0:
     if [[ -f ".git/hooks/pre-push.backup" ]]; then
       bash .git/hooks/pre-push.backup "$@"
     fi
     ```

3. **Test merged hook**:
   ```bash
   bash .git/hooks/pre-push
   ```

### From Husky or Other Tools

If using Husky, lint-staged, or similar:

1. **Keep existing tools**: These scripts are complementary
2. **Run in sequence**: Add validation to Husky config
   ```json
   {
     "husky": {
       "hooks": {
         "pre-push": "bash scripts/validate-repository.sh && lint-staged"
       }
     }
   }
   ```

3. **Or replace**: Remove Husky, use native git hooks
   ```bash
   npm uninstall husky
   bash scripts/install-git-hooks.sh
   ```

---

## Performance Notes

- **Pre-push hook**: ~5-15 seconds depending on repository size
- **Validation suite**: ~10-30 seconds depending on markdown file count
- **Markdown linting**: Largest time cost, can be disabled if too slow

**Optimization Tips**:
1. Use `.validation_exceptions` to skip large/generated files
2. Run full validation in CI, lighter checks in hooks
3. Cache markdownlint results in CI (GitHub Actions cache)

---

## Support and Customization

These scripts are designed to be:
- **Self-contained**: No external dependencies except optional markdownlint
- **Well-commented**: Clear customization points marked
- **Defensive**: Graceful degradation if tools missing
- **Portable**: Works on Linux, macOS, WSL

For issues or customization help, see the main SKILL.md documentation.
