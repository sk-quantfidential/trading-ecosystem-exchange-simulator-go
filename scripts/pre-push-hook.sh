#!/bin/bash
# Pre-push git hook for claude-defaults repository
# Validates branch names, PR documentation, and commit standards before push
#
# Installation:
#   cp scripts/pre-push-hook.sh .git/hooks/pre-push
#   chmod +x .git/hooks/pre-push

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}🔍 Running pre-push validation checks...${NC}"
echo ""

# Get current branch
CURRENT_BRANCH=$(git branch --show-current)
echo -e "${BLUE}Current branch: ${CURRENT_BRANCH}${NC}"

# ============================================================================
# CHECK 1: Verify NOT on protected branch (main/master)
# ============================================================================
echo ""
echo -e "${YELLOW}[1/6] Checking protected branch...${NC}"

if [[ "$CURRENT_BRANCH" == "main" ]] || [[ "$CURRENT_BRANCH" == "master" ]]; then
  echo -e "${RED}❌ ERROR: Cannot push from protected branch: $CURRENT_BRANCH${NC}"
  echo ""
  echo "You should never push directly from main/master."
  echo "Create a feature branch instead:"
  echo "  git checkout -b feature/epic-XXX-9999-milestone-behavior"
  echo ""
  exit 1
fi

echo -e "${GREEN}✅ Not on protected branch${NC}"

# ============================================================================
# CHECK 2: Verify branch name follows convention
# ============================================================================
echo ""
echo -e "${YELLOW}[2/6] Checking branch name convention...${NC}"

# Valid branch types: feature, fix, docs, style, refactor, test, chore, ci
BRANCH_REGEX="^(feature|fix|docs|style|refactor|test|chore|ci)/epic-[A-Z]{3}-[0-9]{4}-.+"

if ! [[ "$CURRENT_BRANCH" =~ $BRANCH_REGEX ]]; then
  echo -e "${RED}❌ ERROR: Branch name doesn't follow convention${NC}"
  echo ""
  echo "Expected format: type/epic-XXX-9999-milestone-behavior"
  echo "Valid types: feature, fix, docs, style, refactor, test, chore, ci"
  echo ""
  echo "Examples:"
  echo "  feature/epic-CLD-0013-cleanup-repository-housekeeping"
  echo "  fix/epic-API-0042-auth-token-expiry-bug"
  echo "  chore/epic-WEB-0015-deps-upgrade-typescript"
  echo ""
  echo "Current branch: $CURRENT_BRANCH"
  echo ""
  exit 1
fi

echo -e "${GREEN}✅ Branch name follows convention${NC}"

# Extract epic info from branch name
BRANCH_TYPE=$(echo "$CURRENT_BRANCH" | cut -d'/' -f1)
EPIC_INFO=$(echo "$CURRENT_BRANCH" | grep -oE "epic-[A-Z]{3}-[0-9]{4}" || echo "")

# ============================================================================
# CHECK 3: Verify PR documentation exists
# ============================================================================
echo ""
echo -e "${YELLOW}[3/6] Checking for PR documentation...${NC}"

# Look for PR file matching branch name (with slashes converted to dashes)
BRANCH_FILENAME=$(echo "$CURRENT_BRANCH" | sed 's/\//-/g')
PR_FILE=""

if [[ -d "docs/prs" ]] && [[ -f "docs/prs/${BRANCH_FILENAME}.md" ]]; then
  PR_FILE="docs/prs/${BRANCH_FILENAME}.md"
fi

if [[ -z "$PR_FILE" ]]; then
  echo -e "${RED}❌ ERROR: No PR documentation found in docs/prs/${NC}"
  echo ""
  echo "Expected PR file:"
  echo "  docs/prs/${BRANCH_FILENAME}.md"
  echo ""
  echo "Create PR documentation before pushing:"
  echo "  1. Copy template: cp docs/PULL_REQUEST_TEMPLATE.md docs/prs/${BRANCH_FILENAME}.md"
  echo "  2. Fill in PR details (summary, changes, testing, etc.)"
  echo "  3. Commit: git add docs/prs/ && git commit -m 'docs: add PR documentation'"
  echo "  4. Try push again"
  echo ""

  # List existing PR files to help user
  if [[ -d "docs/prs" ]] && [[ -n "$(ls -A docs/prs/*.md 2>/dev/null)" ]]; then
    echo "Existing PR files:"
    ls -1 docs/prs/*.md | sed 's/^/  /'
    echo ""
  fi

  exit 1
fi

echo -e "${GREEN}✅ Found PR documentation: ${PR_FILE}${NC}"

# ============================================================================
# CHECK 4: Verify PR file has required sections
# ============================================================================
echo ""
echo -e "${YELLOW}[4/6] Validating PR documentation content...${NC}"

PR_WARNINGS=()

# Check for required sections (standardized across all tools)
# Required: ## Summary, ## Testing (or ## Quality Assurance), ## What Changed
if ! grep -q "## Summary" "$PR_FILE"; then
  PR_WARNINGS+=("Missing '## Summary' section")
fi

if ! grep -q "## Testing" "$PR_FILE" && ! grep -q "## Quality Assurance" "$PR_FILE"; then
  PR_WARNINGS+=("Missing '## Testing' or '## Quality Assurance' section")
fi

if ! grep -q "## Files Changed" "$PR_FILE" && ! grep -q "## What Changed" "$PR_FILE"; then
  PR_WARNINGS+=("Missing '## Files Changed' or '## What Changed' section")
fi

# Check for epic reference in PR file
if [[ -n "$EPIC_INFO" ]]; then
  if ! grep -qi "$EPIC_INFO" "$PR_FILE"; then
    PR_WARNINGS+=("Epic reference '$EPIC_INFO' not found in PR file")
  fi
fi

# Check for placeholder text (common in templates)
if grep -q "PLACEHOLDER" "$PR_FILE" || grep -q "TODO:" "$PR_FILE" || grep -q "FIXME:" "$PR_FILE"; then
  PR_WARNINGS+=("Found placeholder text (PLACEHOLDER/TODO/FIXME) - ensure PR is complete")
fi

if [[ ${#PR_WARNINGS[@]} -gt 0 ]]; then
  echo -e "${YELLOW}⚠️  PR documentation warnings:${NC}"
  for warning in "${PR_WARNINGS[@]}"; do
    echo -e "  ${YELLOW}•${NC} $warning"
  done
  echo ""
  echo "PR file: $PR_FILE"
  echo ""
  read -p "Continue with push anyway? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Push cancelled. Fix PR documentation and try again.${NC}"
    exit 1
  fi
else
  echo -e "${GREEN}✅ PR documentation has required sections${NC}"
fi

# ============================================================================
# CHECK 5: Verify TODO.md or TODO-MASTER.md was updated
# ============================================================================
echo ""
echo -e "${YELLOW}[5/6] Checking TODO documentation updates...${NC}"

# Determine which TODO file to check for
TODO_FILE=""
if [[ -f "TODO-MASTER.md" ]]; then
  TODO_FILE="TODO-MASTER.md"
elif [[ -f "TODO.md" ]]; then
  TODO_FILE="TODO.md"
fi

if [[ -z "$TODO_FILE" ]]; then
  echo -e "${YELLOW}⚠️  No TODO.md or TODO-MASTER.md found${NC}"
  echo ""
  read -p "Continue with push anyway? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Push cancelled.${NC}"
    exit 1
  fi
else
  # Get commits that will be pushed
  REMOTE_BRANCH="origin/$CURRENT_BRANCH"
  if git rev-parse --verify "$REMOTE_BRANCH" >/dev/null 2>&1; then
    # Branch exists on remote, check new commits
    NEW_COMMITS=$(git log "$REMOTE_BRANCH..HEAD" --oneline)

    if [[ -n "$NEW_COMMITS" ]]; then
      # Check if TODO file was modified in any of the new commits
      TODO_MODIFIED=$(git log "$REMOTE_BRANCH..HEAD" --name-only --oneline | grep -E "(TODO\.md|TODO-MASTER\.md)" | wc -l || true)

      if [[ $TODO_MODIFIED -eq 0 ]]; then
        echo -e "${YELLOW}⚠️  $TODO_FILE not modified in new commits${NC}"
        echo ""
        echo "New commits to push:"
        echo "$NEW_COMMITS" | sed 's/^/  /'
        echo ""
        echo "Consider updating $TODO_FILE if this work completes tasks or milestones."
        echo ""
        read -p "Continue with push anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
          echo -e "${RED}Push cancelled. Update $TODO_FILE and try again.${NC}"
          exit 1
        fi
      else
        echo -e "${GREEN}✅ TODO documentation was updated ($TODO_MODIFIED commit(s))${NC}"
      fi
    fi
  else
    # First push of this branch
    echo -e "${BLUE}ℹ️  First push of branch (no remote tracking yet)${NC}"

    # Check if TODO file exists in commits
    TODO_IN_BRANCH=$(git log --name-only --oneline | grep -E "(TODO\.md|TODO-MASTER\.md)" | wc -l || true)
    if [[ $TODO_IN_BRANCH -gt 0 ]]; then
      echo -e "${GREEN}✅ TODO documentation included in branch commits${NC}"
    else
      echo -e "${YELLOW}⚠️  $TODO_FILE not found in branch commits${NC}"
      echo ""
      read -p "Continue with push anyway? (y/N) " -n 1 -r
      echo
      if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Push cancelled.${NC}"
        exit 1
      fi
    fi
  fi
fi

# ============================================================================
# CHECK 6: Markdown linting
# ============================================================================
echo ""
echo -e "${YELLOW}[6/7] Checking markdown files...${NC}"

# Check if markdownlint is installed
if command -v markdownlint &> /dev/null; then
  # Find markdown files in repository (excluding node_modules, vendor, .claude/*, etc.)
  MARKDOWN_FILES=$(find . -name "*.md" \
    -not -path "*/node_modules/*" \
    -not -path "*/vendor/*" \
    -not -path "*/.git/*" \
    -not -path "*/.claude/*" \
    -not -path "*/dist/*" \
    -not -path "*/build/*" \
    2>/dev/null || true)

  if [[ -n "$MARKDOWN_FILES" ]]; then
    # Run markdownlint with config if available
    MARKDOWNLINT_CONFIG=""
    if [[ -f ".markdownlint.json" ]]; then
      MARKDOWNLINT_CONFIG="--config .markdownlint.json"
    fi

    # Capture markdownlint output
    MARKDOWN_ERRORS=$(echo "$MARKDOWN_FILES" | xargs markdownlint $MARKDOWNLINT_CONFIG 2>&1 || true)

    if [[ -n "$MARKDOWN_ERRORS" ]]; then
      echo -e "${RED}❌ Markdown linting errors found${NC}"
      echo ""
      echo "$MARKDOWN_ERRORS"
      echo ""
      echo "To fix automatically:"
      echo "  markdownlint --fix *.md docs/**/*.md"
      echo ""
      echo "To configure rules, edit .markdownlint.json"
      echo ""
      read -p "Continue with push anyway? (y/N) " -n 1 -r
      echo
      if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Push cancelled. Fix markdown errors and try again.${NC}"
        exit 1
      fi
      echo -e "${YELLOW}⚠️  Proceeding despite markdown errors${NC}"
    else
      echo -e "${GREEN}✅ All markdown files pass linting${NC}"
    fi
  else
    echo -e "${BLUE}ℹ️  No markdown files found${NC}"
  fi
else
  echo -e "${YELLOW}⚠️  markdownlint not installed - skipping markdown validation${NC}"
  echo -e "${BLUE}ℹ️  Install with: npm install -g markdownlint-cli${NC}"
fi

# ============================================================================
# CHECK 7: Run full validation suite
# ============================================================================
echo ""
echo -e "${YELLOW}[7/7] Running full validation suite...${NC}"
echo -e "${BLUE}ℹ️  This may take a moment - validating markdown, cross-references, and code blocks${NC}"
echo ""

# Check if validate-all.sh exists
if [[ ! -f "scripts/validate-all.sh" ]]; then
  echo -e "${YELLOW}⚠️  Validation script not found: scripts/validate-all.sh${NC}"
  echo -e "${BLUE}ℹ️  Skipping validation suite${NC}"
else
  # Run validation suite and capture output
  VALIDATION_OUTPUT=$(bash scripts/validate-all.sh 2>&1)
  VALIDATION_EXIT_CODE=$?

  if [[ $VALIDATION_EXIT_CODE -eq 0 ]]; then
    echo -e "${GREEN}✅ Full validation suite passed${NC}"
    # Show summary lines only (lines with ✅ or checkmarks)
    echo "$VALIDATION_OUTPUT" | grep -E "(✅|Cross-reference|validation)" | head -5 || true
  else
    echo -e "${RED}❌ Validation suite failed${NC}"
    echo ""
    echo "Validation errors:"
    echo "$VALIDATION_OUTPUT" | head -30
    if [[ $(echo "$VALIDATION_OUTPUT" | wc -l) -gt 30 ]]; then
      echo "... (showing first 30 lines)"
    fi
    echo ""
    echo -e "${RED}Please fix validation errors before pushing.${NC}"
    echo ""
    echo "To run validations manually:"
    echo "  bash scripts/validate-all.sh"
    echo ""
    read -p "Continue with push anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo -e "${RED}Push cancelled. Fix validation errors and try again.${NC}"
      exit 1
    fi
    echo -e "${YELLOW}⚠️  Proceeding despite validation errors${NC}"
  fi
fi

# ============================================================================
# SUMMARY
# ============================================================================
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ All pre-push validation checks passed!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}Summary:${NC}"
echo -e "  Branch: ${CURRENT_BRANCH}"
echo -e "  PR Prefix: ${PR_PREFIX}"
echo -e "  PR File: ${PR_FILE}"
echo ""
echo -e "${GREEN}Proceeding with push to remote...${NC}"
echo ""

# Exit 0 allows the push to proceed
exit 0
