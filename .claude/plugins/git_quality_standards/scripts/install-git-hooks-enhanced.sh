#!/bin/bash
# Enhanced git hooks installer with template support
# Installs hooks and optionally copies GitHub Actions workflows and PR templates

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo -e "${BLUE}Installing git hooks and optional templates...${NC}"
echo ""

# Check we're in a git repository
if [[ ! -d ".git" ]]; then
  echo -e "${YELLOW}❌ Error: Not in a git repository root${NC}"
  echo "Run this script from the repository root directory."
  exit 1
fi

# Check scripts directory exists
SCRIPT_DIR=".claude/plugins/git_quality_standards/scripts"
if [[ ! -d "$SCRIPT_DIR" ]]; then
  echo -e "${YELLOW}❌ Error: $SCRIPT_DIR directory not found${NC}"
  echo "Ensure git_quality_standards plugin is installed."
  exit 1
fi

# Install pre-push hook
if [[ -f "$SCRIPT_DIR/pre-push-hook.sh" ]]; then
  echo -e "${BLUE}Installing pre-push hook...${NC}"
  cp "$SCRIPT_DIR/pre-push-hook.sh" .git/hooks/pre-push
  chmod +x .git/hooks/pre-push
  echo -e "${GREEN}✅ Pre-push hook installed${NC}"
else
  echo -e "${YELLOW}⚠️  $SCRIPT_DIR/pre-push-hook.sh not found, skipping${NC}"
fi

# Optional: Install GitHub Actions workflows
INSTALL_GH_ACTIONS="${1:-prompt}"

if [[ "$INSTALL_GH_ACTIONS" == "prompt" ]]; then
  echo ""
  read -p "Install GitHub Actions workflows? (y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    INSTALL_GH_ACTIONS="yes"
  else
    INSTALL_GH_ACTIONS="no"
  fi
fi

if [[ "$INSTALL_GH_ACTIONS" == "yes" ]] || [[ "$INSTALL_GH_ACTIONS" == "-y" ]]; then
  WORKFLOW_DIR=".claude/plugins/git_quality_standards/workflows"

  if [[ -d "$WORKFLOW_DIR" ]]; then
    echo ""
    echo -e "${BLUE}Installing GitHub Actions workflows...${NC}"
    mkdir -p .github/workflows

    if [[ -f "$WORKFLOW_DIR/pr-checks.yml" ]]; then
      cp "$WORKFLOW_DIR/pr-checks.yml" .github/workflows/pr-checks.yml
      echo -e "${GREEN}✅ Created .github/workflows/pr-checks.yml${NC}"
      echo -e "${YELLOW}   Remember to customize project codes (XXX → YOUR_CODE)${NC}"
    fi

    if [[ -f "$WORKFLOW_DIR/validation.yml" ]]; then
      cp "$WORKFLOW_DIR/validation.yml" .github/workflows/validation.yml
      echo -e "${GREEN}✅ Created .github/workflows/validation.yml${NC}"
    fi
  else
    echo -e "${YELLOW}⚠️  Workflow directory not found: $WORKFLOW_DIR${NC}"
  fi
fi

# Optional: Install PR template
INSTALL_PR_TEMPLATE="${2:-prompt}"

if [[ "$INSTALL_PR_TEMPLATE" == "prompt" ]]; then
  echo ""
  read -p "Install PR documentation template? (y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    INSTALL_PR_TEMPLATE="yes"
  else
    INSTALL_PR_TEMPLATE="no"
  fi
fi

if [[ "$INSTALL_PR_TEMPLATE" == "yes" ]] || [[ "$INSTALL_PR_TEMPLATE" == "-y" ]]; then
  TEMPLATE_DIR=".claude/plugins/git_quality_standards/templates"

  if [[ -f "$TEMPLATE_DIR/pull_request_template.md" ]]; then
    echo ""
    echo -e "${BLUE}Installing PR template...${NC}"
    mkdir -p .github
    cp "$TEMPLATE_DIR/pull_request_template.md" .github/pull_request_template.md
    echo -e "${GREEN}✅ Created .github/pull_request_template.md${NC}"
    echo -e "${YELLOW}   This appears in GitHub when creating PRs${NC}"
  else
    echo -e "${YELLOW}⚠️  PR template not found${NC}"
  fi
fi

# Optional: Install validation exceptions template
if [[ ! -f ".validation_exceptions" ]]; then
  INSTALL_VALIDATION="${3:-no}"

  if [[ "$INSTALL_VALIDATION" == "prompt" ]]; then
    echo ""
    read -p "Create .validation_exceptions file? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      INSTALL_VALIDATION="yes"
    else
      INSTALL_VALIDATION="no"
    fi
  fi

  if [[ "$INSTALL_VALIDATION" == "yes" ]] || [[ "$INSTALL_VALIDATION" == "-y" ]]; then
    TEMPLATE_DIR=".claude/plugins/git_quality_standards/templates"

    if [[ -f "$TEMPLATE_DIR/.validation_exceptions.template" ]]; then
      echo ""
      echo -e "${BLUE}Creating validation exceptions file...${NC}"
      cp "$TEMPLATE_DIR/.validation_exceptions.template" .validation_exceptions
      echo -e "${GREEN}✅ Created .validation_exceptions${NC}"
      echo -e "${YELLOW}   Customize this file to exclude files from validation${NC}"
    fi
  fi
fi

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Git hooks installation complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}Installed hooks:${NC}"
echo "  • pre-push: Validates branch name, PR documentation, markdown, TODO.md updates"
echo ""
echo -e "${BLUE}What these hooks do:${NC}"
echo "  1. Prevent pushing from main/master branches"
echo "  2. Validate branch naming convention"
echo "  3. Ensure PR documentation exists in docs/prs/"
echo "  4. Check PR file has required sections"
echo "  5. Verify TODO.md was updated"
echo "  6. Lint markdown files with markdownlint"
echo "  7. Run full validation suite (if available)"
echo ""
echo -e "${YELLOW}Note: Hooks run automatically before git push${NC}"
echo "To skip hooks temporarily: git push --no-verify"
echo ""
echo -e "${BLUE}Optional files created:${NC}"
[[ -f ".github/workflows/pr-checks.yml" ]] && echo "  • .github/workflows/pr-checks.yml"
[[ -f ".github/workflows/validation.yml" ]] && echo "  • .github/workflows/validation.yml"
[[ -f ".github/pull_request_template.md" ]] && echo "  • .github/pull_request_template.md"
[[ -f ".validation_exceptions" ]] && echo "  • .validation_exceptions"
echo ""
echo -e "${BLUE}For PR automation, use:${NC}"
echo "  scripts/create-pr.sh    # Creates GitHub PR with auto-populated description"
echo ""
