#!/bin/bash
# Install git hooks for claude-defaults repository
# This script copies hook scripts to .git/hooks/ and makes them executable

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo -e "${BLUE}Installing git hooks for claude-defaults...${NC}"
echo ""

# Check we're in a git repository
if [[ ! -d ".git" ]]; then
  echo -e "${YELLOW}❌ Error: Not in a git repository root${NC}"
  echo "Run this script from the repository root directory."
  exit 1
fi

# Check scripts directory exists
if [[ ! -d "scripts" ]]; then
  echo -e "${YELLOW}❌ Error: scripts/ directory not found${NC}"
  echo "This doesn't appear to be the claude-defaults repository."
  exit 1
fi

# Install pre-push hook
if [[ -f "scripts/pre-push-hook.sh" ]]; then
  echo -e "${BLUE}Installing pre-push hook...${NC}"
  cp scripts/pre-push-hook.sh .git/hooks/pre-push
  chmod +x .git/hooks/pre-push
  echo -e "${GREEN}✅ Pre-push hook installed${NC}"
else
  echo -e "${YELLOW}⚠️  scripts/pre-push-hook.sh not found, skipping${NC}"
fi

# Future hooks can be added here
# if [[ -f "scripts/pre-commit-hook.sh" ]]; then
#   echo -e "${BLUE}Installing pre-commit hook...${NC}"
#   cp scripts/pre-commit-hook.sh .git/hooks/pre-commit
#   chmod +x .git/hooks/pre-commit
#   echo -e "${GREEN}✅ Pre-commit hook installed${NC}"
# fi

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Git hooks installation complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}Installed hooks:${NC}"
echo "  • pre-push: Validates branch name, PR documentation, TODO.md updates"
echo ""
echo -e "${BLUE}What these hooks do:${NC}"
echo "  1. Prevent pushing from main/master branches"
echo "  2. Validate branch naming convention"
echo "  3. Ensure PR documentation exists in docs/prs/"
echo "  4. Check PR file has required sections"
echo "  5. Verify TODO.md was updated"
echo ""
echo -e "${YELLOW}Note: Hooks run automatically before git push${NC}"
echo "To skip hooks temporarily: git push --no-verify"
echo ""
