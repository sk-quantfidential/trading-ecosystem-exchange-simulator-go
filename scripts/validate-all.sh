#!/bin/bash
# Full validation suite for repository structure and documentation
# Validates markdown, cross-references, and repository standards

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

VALIDATION_ERRORS=0

echo ""
echo -e "${BLUE}🔍 Running full repository validation suite...${NC}"
echo ""

# ============================================================================
# CHECK 1: Required files exist
# ============================================================================
echo -e "${YELLOW}[1/6] Checking required files...${NC}"

REQUIRED_FILES=(
  "README.md"
  "TODO.md"
  "CONTRIBUTING.md"
  ".gitignore"
  ".validation_exceptions"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo -e "${RED}❌ Missing required file: $file${NC}"
    VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
  fi
done

if [[ $VALIDATION_ERRORS -eq 0 ]]; then
  echo -e "${GREEN}✅ All required files present${NC}"
fi

# ============================================================================
# CHECK 2: Git quality standards plugin
# ============================================================================
echo ""
echo -e "${YELLOW}[2/6] Checking git quality standards plugin...${NC}"

PLUGIN_DIR=".claude/plugins/git_quality_standards"
REQUIRED_PLUGIN_FILES=(
  "$PLUGIN_DIR/README.md"
  "$PLUGIN_DIR/scripts/pre-push-hook.sh"
  "$PLUGIN_DIR/scripts/install-git-hooks-enhanced.sh"
)

for file in "${REQUIRED_PLUGIN_FILES[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo -e "${RED}❌ Missing plugin file: $file${NC}"
    VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
  fi
done

if [[ $VALIDATION_ERRORS -eq 0 ]]; then
  echo -e "${GREEN}✅ Git quality standards plugin present${NC}"
fi

# ============================================================================
# CHECK 3: PR documentation exists for current branch
# ============================================================================
echo ""
echo -e "${YELLOW}[3/6] Checking for PR documentation...${NC}"

# Get current branch name
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")

if [[ -z "$CURRENT_BRANCH" ]] || [[ "$CURRENT_BRANCH" == "main" ]] || [[ "$CURRENT_BRANCH" == "master" ]]; then
  echo -e "${YELLOW}⚠️  On main/master branch - PR documentation check skipped${NC}"
else
  # Extract branch components
  if [[ "$CURRENT_BRANCH" =~ ^([^/]+)/epic-([A-Z]{3})-([0-9]{4})(-(.+))?$ ]]; then
    BRANCH_TYPE="${BASH_REMATCH[1]}"
    EPIC_CODE="${BASH_REMATCH[2]}"
    EPIC_NUM="${BASH_REMATCH[4]}"
    EPIC_INFO="epic-${EPIC_CODE}-${BASH_REMATCH[3]}"

    # Convert branch type to PR prefix
    case "$BRANCH_TYPE" in
      feature) PR_PREFIX="feat" ;;
      fix) PR_PREFIX="fix" ;;
      docs) PR_PREFIX="docs" ;;
      style) PR_PREFIX="style" ;;
      refactor) PR_PREFIX="refac" ;;
      test) PR_PREFIX="test" ;;
      chore) PR_PREFIX="chore" ;;
      ci) PR_PREFIX="ci" ;;
      *) PR_PREFIX="$BRANCH_TYPE" ;;
    esac

    # Look for PR documentation matching current branch
    PR_FILES_FOUND=()
    if [[ -d "docs/prs" ]]; then
      # Pattern 1: {prefix}-epic-XXX-9999-*.md
      while IFS= read -r -d '' file; do
        PR_FILES_FOUND+=("$file")
      done < <(find docs/prs -name "${PR_PREFIX}-${EPIC_INFO}-*.md" -print0 2>/dev/null || true)

      # Pattern 2: Full branch name with slashes replaced
      BRANCH_FILENAME=$(echo "$CURRENT_BRANCH" | sed 's/\//-/g')
      if [[ -f "docs/prs/${BRANCH_FILENAME}.md" ]]; then
        PR_FILES_FOUND+=("docs/prs/${BRANCH_FILENAME}.md")
      fi
    fi

    if [[ ${#PR_FILES_FOUND[@]} -eq 0 ]]; then
      echo -e "${RED}❌ ERROR: No PR documentation found for current branch${NC}"
      echo -e "${BLUE}   Branch: $CURRENT_BRANCH${NC}"
      echo -e "${BLUE}   Expected PR file matching one of:${NC}"
      echo -e "${BLUE}     docs/prs/${PR_PREFIX}-${EPIC_INFO}-*.md${NC}"
      echo -e "${BLUE}     docs/prs/${BRANCH_FILENAME}.md${NC}"
      echo ""
      echo -e "${BLUE}   Create PR documentation before pushing:${NC}"
      echo -e "${BLUE}     mkdir -p docs/prs${NC}"
      echo -e "${BLUE}     # Create and fill in PR documentation${NC}"
      echo -e "${BLUE}     git add docs/prs/ && git commit -m 'docs: add PR documentation'${NC}"
      VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))

      # List existing PR files to help user
      if [[ -d "docs/prs" ]] && [[ -n "$(ls -A docs/prs/*.md 2>/dev/null)" ]]; then
        echo ""
        echo -e "${BLUE}   Existing PR files:${NC}"
        ls -1 docs/prs/*.md 2>/dev/null | sed 's/^/     /' || true
      fi
    else
      echo -e "${GREEN}✅ Found PR documentation: ${PR_FILES_FOUND[0]}${NC}"

      # ============================================================================
      # CHECK 4: Validate PR documentation content
      # ============================================================================
      echo ""
      echo -e "${YELLOW}[4/7] Validating PR documentation content...${NC}"

      PR_FILE="${PR_FILES_FOUND[0]}"
      PR_WARNINGS=()

      # Check for required sections
      if ! grep -q "## Summary" "$PR_FILE"; then
        PR_WARNINGS+=("Missing '## Summary' section")
      fi

      if ! grep -q "## Quality Assurance" "$PR_FILE" && ! grep -q "## Testing" "$PR_FILE"; then
        PR_WARNINGS+=("Missing '## Quality Assurance' or '## Testing' section")
      fi

      if ! grep -q "## Files Changed" "$PR_FILE" && ! grep -q "## What Changed" "$PR_FILE" && ! grep -q "## Changes" "$PR_FILE"; then
        PR_WARNINGS+=("Missing '## Files Changed', '## What Changed', or '## Changes' section")
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
        echo -e "${RED}❌ PR documentation content issues:${NC}"
        for warning in "${PR_WARNINGS[@]}"; do
          echo -e "  ${RED}•${NC} $warning"
        done
        echo ""
        echo -e "${BLUE}   PR file: $PR_FILE${NC}"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
      else
        echo -e "${GREEN}✅ PR documentation has required sections${NC}"
      fi
    fi
  else
    echo -e "${YELLOW}⚠️  Branch doesn't follow epic naming convention - PR check skipped${NC}"
    echo -e "${BLUE}   Expected: type/epic-XXX-9999-description${NC}"
    echo -e "${BLUE}   Actual: $CURRENT_BRANCH${NC}"
  fi
fi

# ============================================================================
# CHECK 5: GitHub Actions workflows
# ============================================================================
echo ""
echo -e "${YELLOW}[5/7] Checking GitHub Actions workflows...${NC}"

WORKFLOW_FILES=(
  ".github/workflows/pr-checks.yml"
  ".github/workflows/validation.yml"
  ".github/pull_request_template.md"
)

for file in "${WORKFLOW_FILES[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo -e "${YELLOW}⚠️  Optional file missing: $file${NC}"
  fi
done

if [[ -f ".github/workflows/pr-checks.yml" ]]; then
  echo -e "${GREEN}✅ GitHub Actions workflows configured${NC}"
else
  echo -e "${YELLOW}⚠️  GitHub Actions not configured (optional)${NC}"
fi

# ============================================================================
# CHECK 6: Documentation structure
# ============================================================================
echo ""
echo -e "${YELLOW}[6/7] Checking documentation structure...${NC}"

# Just verify docs directory structure exists
if [[ -d "docs" ]]; then
  echo -e "${GREEN}✅ Documentation directory structure present${NC}"
else
  echo -e "${YELLOW}⚠️  No docs/ directory (will be created when needed)${NC}"
fi

# ============================================================================
# CHECK 7: Markdown linting configuration
# ============================================================================
echo ""
echo -e "${YELLOW}[7/7] Checking markdown linting configuration...${NC}"

if [[ ! -f ".markdownlint.json" ]]; then
  echo -e "${YELLOW}⚠️  .markdownlint.json not found (optional)${NC}"
else
  echo -e "${GREEN}✅ Markdown linting configured${NC}"
fi

# Validate markdown if markdownlint is available
if command -v markdownlint &> /dev/null; then
  MARKDOWN_FILES=$(find . -name "*.md" \
    -not -path "*/node_modules/*" \
    -not -path "*/vendor/*" \
    -not -path "*/.git/*" \
    -not -path "*/.claude/*" \
    -not -path "*/dist/*" \
    -not -path "*/build/*" \
    2>/dev/null || true)

  if [[ -n "$MARKDOWN_FILES" ]]; then
    MARKDOWNLINT_CONFIG=""
    if [[ -f ".markdownlint.json" ]]; then
      MARKDOWNLINT_CONFIG="--config .markdownlint.json"
    fi

    MARKDOWN_ERRORS=$(echo "$MARKDOWN_FILES" | xargs markdownlint $MARKDOWNLINT_CONFIG 2>&1 || true)

    if [[ -n "$MARKDOWN_ERRORS" ]]; then
      echo -e "${RED}❌ Markdown linting errors found${NC}"
      echo "$MARKDOWN_ERRORS" | head -10
      VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
    else
      echo -e "${GREEN}✅ All markdown files valid${NC}"
    fi
  fi
else
  echo -e "${BLUE}ℹ️  markdownlint not installed, skipping markdown validation${NC}"
fi

# ============================================================================
# SUMMARY
# ============================================================================
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [[ $VALIDATION_ERRORS -eq 0 ]]; then
  echo -e "${GREEN}✅ All validation checks passed!${NC}"
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  exit 0
else
  echo -e "${RED}❌ Validation failed with $VALIDATION_ERRORS error(s)${NC}"
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  exit 1
fi
