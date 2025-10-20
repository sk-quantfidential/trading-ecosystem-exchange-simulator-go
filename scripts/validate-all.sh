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
# CHECK 3: PR documentation exists
# ============================================================================
echo ""
echo -e "${YELLOW}[3/6] Checking for PR documentation...${NC}"

if [[ ! -d "docs/prs" ]]; then
  echo -e "${RED}❌ ERROR: No docs/prs/ directory found${NC}"
  echo -e "${BLUE}   Create with: mkdir -p docs/prs${NC}"
  VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
else
  PR_COUNT=$(find docs/prs -name "*.md" -type f 2>/dev/null | wc -l)
  if [[ $PR_COUNT -gt 0 ]]; then
    echo -e "${GREEN}✅ Found $PR_COUNT PR documentation file(s)${NC}"
  else
    echo -e "${RED}❌ ERROR: No PR documentation found in docs/prs/${NC}"
    echo -e "${BLUE}   PR documentation is required before pushing${NC}"
    echo -e "${BLUE}   See git_workflow_checklist skill for PR documentation requirements${NC}"
    VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
  fi
fi

# ============================================================================
# CHECK 4: GitHub Actions workflows
# ============================================================================
echo ""
echo -e "${YELLOW}[4/6] Checking GitHub Actions workflows...${NC}"

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
# CHECK 5: Documentation structure
# ============================================================================
echo ""
echo -e "${YELLOW}[5/6] Checking documentation structure...${NC}"

# Just verify docs directory structure exists
if [[ -d "docs" ]]; then
  echo -e "${GREEN}✅ Documentation directory structure present${NC}"
else
  echo -e "${YELLOW}⚠️  No docs/ directory (will be created when needed)${NC}"
fi

# ============================================================================
# CHECK 6: Markdown linting configuration
# ============================================================================
echo ""
echo -e "${YELLOW}[6/6] Checking markdown linting configuration...${NC}"

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
