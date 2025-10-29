#!/bin/bash
# Automated PR creation with docs/prs/ content as description
# Uses gh CLI to create GitHub PR with auto-populated title and body

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo ""
echo -e "${BLUE}🚀 Automated PR Creation${NC}"
echo ""

# Check gh CLI is installed
if ! command -v gh &> /dev/null; then
  echo -e "${RED}❌ Error: GitHub CLI (gh) is not installed${NC}"
  echo ""
  echo "Install it with:"
  echo "  macOS:   brew install gh"
  echo "  Linux:   See https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
  echo "  Windows: See https://github.com/cli/cli#installation"
  echo ""
  exit 1
fi

# Check gh is authenticated
if ! gh auth status &> /dev/null; then
  echo -e "${YELLOW}⚠️  Not authenticated with GitHub${NC}"
  echo "Authenticate with: gh auth login"
  exit 1
fi

# Get current branch
BRANCH=$(git branch --show-current)

if [[ -z "$BRANCH" ]]; then
  echo -e "${RED}❌ Error: Not on a branch${NC}"
  exit 1
fi

echo -e "${BLUE}Current branch: ${BRANCH}${NC}"

# Check branch naming convention
if ! [[ "$BRANCH" =~ ^(feature|fix|docs|style|refactor|test|chore|ci)/epic-[A-Z]{3}-[0-9]{4}-.+ ]]; then
  echo -e "${YELLOW}⚠️  Warning: Branch name doesn't follow convention${NC}"
  echo "Expected: type/epic-XXX-9999-milestone-behavior"
  echo ""
  read -p "Continue anyway? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

# Find PR documentation file
PR_DOC=""

# Try exact match: docs/prs/{branch-name}.md
if [[ -f "docs/prs/${BRANCH}.md" ]]; then
  PR_DOC="docs/prs/${BRANCH}.md"
fi

# Try with slashes converted to dashes (common PR file naming convention)
if [[ -z "$PR_DOC" ]]; then
  BRANCH_WITH_DASHES="${BRANCH//\//-}"
  if [[ -f "docs/prs/${BRANCH_WITH_DASHES}.md" ]]; then
    PR_DOC="docs/prs/${BRANCH_WITH_DASHES}.md"
  fi
fi

# If still not found, list all and let user choose
if [[ -z "$PR_DOC" ]]; then
  if [[ -d "docs/prs" ]]; then
    ALL_DOCS=($(find docs/prs -name "*.md" 2>/dev/null || true))

    if [[ ${#ALL_DOCS[@]} -eq 0 ]]; then
      echo -e "${RED}❌ Error: No PR documentation found in docs/prs/${NC}"
      echo ""
      echo "Create PR documentation first:"
      echo "  1. Create file: docs/prs/${BRANCH}.md"
      echo "  2. Use template from: ~/.claude/skills/foundations/git_quality_standards/templates/PR_DOCUMENTATION_TEMPLATE.md"
      echo "  3. Run this script again"
      exit 1
    fi

    echo -e "${YELLOW}⚠️  PR doc not found automatically${NC}"
    echo "Available PR documentation:"
    for i in "${!ALL_DOCS[@]}"; do
      echo "  $((i+1)). ${ALL_DOCS[$i]}"
    done
    echo ""
    read -p "Select file (1-${#ALL_DOCS[@]}): " -r
    PR_DOC="${ALL_DOCS[$((REPLY-1))]}"
  else
    echo -e "${RED}❌ Error: docs/prs/ directory not found${NC}"
    exit 1
  fi
fi

echo -e "${GREEN}✅ Found PR documentation: ${PR_DOC}${NC}"
echo ""

# Generate PR title in conventional commit format: type(epic-XXX-9999): description
# Format: {prefix}(epic-XXX-9999): {description}
# Example: chore(epic-TSE-0001): add git quality standards infrastructure

# Extract branch type
BRANCH_TYPE="${BRANCH%%/*}"
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

# Extract epic code (e.g., epic-TSE-0001 from feature/epic-TSE-0001-milestone-description)
if [[ "$BRANCH" =~ (epic-[A-Z]{3}-[0-9]{4}) ]]; then
  EPIC_CODE="${BASH_REMATCH[1]}"
else
  echo -e "${RED}❌ Error: Could not extract epic code from branch${NC}"
  echo "Branch: $BRANCH"
  exit 1
fi

# Extract description (everything after epic-XXX-9999-milestone-)
# Example: feature/epic-TSE-0001-foundation-add-git-quality-standards
# Should extract: "add git quality standards"
BRANCH_SUFFIX="${BRANCH#*/epic-[A-Z][A-Z][A-Z]-[0-9][0-9][0-9][0-9]-}"

# Try more robust extraction if the above didn't work
if [[ "$BRANCH_SUFFIX" == "$BRANCH" ]]; then
  # Fallback: extract everything after the epic pattern
  BRANCH_SUFFIX=$(echo "$BRANCH" | sed -E 's|^[^/]+/epic-[A-Z]{3}-[0-9]{4}-[^-]+-||')
fi

# Convert dashes to spaces and clean up
DESCRIPTION=$(echo "$BRANCH_SUFFIX" | sed 's/-/ /g')

# Construct conventional commit title
PR_TITLE="${PR_PREFIX}(${EPIC_CODE}): ${DESCRIPTION}"

echo -e "${BLUE}PR Title:${NC}"
echo "  $PR_TITLE"
echo ""

# Use PR doc content as body (skip the title line)
PR_BODY=$(tail -n +2 "$PR_DOC")

# Preview
echo -e "${BLUE}PR Body Preview (first 10 lines):${NC}"
echo "$PR_BODY" | head -10
echo "  ..."
echo ""

# Confirm
read -p "Create PR with this title and description? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 0
fi

# Push branch if needed
echo ""
echo -e "${BLUE}Checking if branch is pushed...${NC}"

if ! git ls-remote --heads origin "$BRANCH" | grep -q "$BRANCH"; then
  echo -e "${YELLOW}Branch not on remote, pushing...${NC}"
  git push -u origin "$BRANCH"
else
  echo -e "${GREEN}✅ Branch already on remote${NC}"
fi

# Create PR
echo ""
echo -e "${BLUE}Creating GitHub PR...${NC}"

gh pr create \
  --title "$PR_TITLE" \
  --body "$PR_BODY" \
  --base main

PR_URL=$(gh pr view --json url -q .url)

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Pull Request Created!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}PR URL:${NC} $PR_URL"
echo ""
echo "Next steps:"
echo "  1. Review the PR on GitHub"
echo "  2. Request reviewers"
echo "  3. Wait for CI checks to pass"
echo "  4. Merge when approved"
echo ""
