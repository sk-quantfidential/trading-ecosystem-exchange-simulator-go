#!/bin/bash
# validate-repository.sh - Portable validation script for git quality standards
#
# This is a simplified, portable version suitable for any project adopting
# the git_quality_standards skill. Customize the checks for your project.
#
# Usage:
#   bash scripts/validate-repository.sh
#
# Exit codes:
#   0 - All validations passed (warnings OK)
#   1 - Validation errors found

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
ERRORS=0
WARNINGS=0

echo -e "${BLUE}🔍 Validating Repository Quality Standards...${NC}"
echo ""

# Function to report errors
report_error() {
    echo -e "${RED}✗ $1${NC}"
    ERRORS=$((ERRORS + 1))
}

# Function to report success
report_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to report warnings
report_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
    WARNINGS=$((WARNINGS + 1))
}

# ============================================================================
# CHECK 1: Verify git repository
# ============================================================================
echo -e "${BLUE}Checking git repository status...${NC}"

if [ ! -d ".git" ]; then
    report_error "Not in a git repository root"
    exit 1
fi

report_success "Valid git repository"

# ============================================================================
# CHECK 2: Check for required documentation files
# ============================================================================
echo -e "\n${BLUE}Checking required documentation...${NC}"

# Customize this list for your project
required_docs=("README.md")
optional_docs=("TODO.md" "CLAUDE.md" "CONTRIBUTING.md")

for doc in "${required_docs[@]}"; do
    if [ ! -f "$doc" ]; then
        report_error "Missing required documentation: $doc"
    else
        report_success "Required documentation exists: $doc"
    fi
done

for doc in "${optional_docs[@]}"; do
    if [ ! -f "$doc" ]; then
        report_warning "Optional documentation missing: $doc"
    else
        report_success "Optional documentation exists: $doc"
    fi
done

# ============================================================================
# CHECK 3: Verify branch naming convention (current branch)
# ============================================================================
echo -e "\n${BLUE}Checking current branch name...${NC}"

CURRENT_BRANCH=$(git branch --show-current)

# Skip check if on main/master (expected for initial setup)
if [[ "$CURRENT_BRANCH" == "main" ]] || [[ "$CURRENT_BRANCH" == "master" ]]; then
    echo -e "${BLUE}ℹ️  On protected branch: $CURRENT_BRANCH${NC}"
else
    # Validate branch naming convention
    # Format: type/epic-XXX-9999-milestone-behavior
    BRANCH_REGEX="^(feature|fix|docs|style|refactor|test|chore|ci)/epic-[A-Z]{2,4}-[0-9]{4}-.+"

    if [[ "$CURRENT_BRANCH" =~ $BRANCH_REGEX ]]; then
        report_success "Branch name follows convention: $CURRENT_BRANCH"
    else
        report_warning "Branch name doesn't follow convention: $CURRENT_BRANCH"
        echo -e "${YELLOW}  Expected: type/epic-XXX-9999-milestone-behavior${NC}"
    fi
fi

# ============================================================================
# CHECK 4: Validate Markdown files (if markdownlint available)
# ============================================================================
echo -e "\n${BLUE}Validating Markdown syntax...${NC}"

if command -v markdownlint &> /dev/null; then
    # Build ignore patterns
    MARKDOWNLINT_CMD="markdownlint . --ignore node_modules"

    # Add .validation_exceptions if exists
    if [ -f ".validation_exceptions" ]; then
        while IFS= read -r line; do
            # Skip comments and empty lines
            [[ "$line" =~ ^#.*$ ]] || [ -z "$line" ] && continue

            # Add as ignore pattern
            if [[ "$line" == *"*"* ]]; then
                # Glob pattern - find matching files
                while IFS= read -r -d '' file; do
                    if [ -f "$file" ]; then
                        MARKDOWNLINT_CMD="$MARKDOWNLINT_CMD --ignore \"$file\""
                    fi
                done < <(find . -path "./$line" -type f -print0 2>/dev/null || true)
            else
                # Exact path
                if [ -f "$line" ]; then
                    MARKDOWNLINT_CMD="$MARKDOWNLINT_CMD --ignore \"$line\""
                fi
            fi
        done < .validation_exceptions
    fi

    if eval $MARKDOWNLINT_CMD > /tmp/markdownlint-validate.log 2>&1; then
        report_success "Markdown syntax validation passed"
    else
        report_warning "Markdown formatting issues found"
        echo -e "${YELLOW}First 10 issues:${NC}"
        head -10 /tmp/markdownlint-validate.log | sed 's/^/  /'
        if [ $(wc -l < /tmp/markdownlint-validate.log) -gt 10 ]; then
            echo "  ... (see /tmp/markdownlint-validate.log for full list)"
        fi
    fi
else
    echo -e "${YELLOW}ℹ️  markdownlint not installed, skipping markdown validation${NC}"
    echo -e "${YELLOW}   Install with: npm install -g markdownlint-cli${NC}"
fi

# ============================================================================
# CHECK 5: Verify PR documentation structure (if using docs/prs/)
# ============================================================================
echo -e "\n${BLUE}Checking PR documentation structure...${NC}"

if [ -d "docs/prs" ]; then
    PR_COUNT=$(find docs/prs -name "*.md" -type f 2>/dev/null | wc -l)

    if [ $PR_COUNT -gt 0 ]; then
        report_success "Found $PR_COUNT PR documentation file(s)"

        # Check if current branch has PR docs
        if [[ "$CURRENT_BRANCH" != "main" ]] && [[ "$CURRENT_BRANCH" != "master" ]]; then
            BRANCH_FILENAME=$(echo "$CURRENT_BRANCH" | sed 's/\//-/g')
            if [ -f "docs/prs/${BRANCH_FILENAME}.md" ]; then
                report_success "PR documentation exists for current branch"
            else
                # Extract epic info and check for matching PR files
                EPIC_INFO=$(echo "$CURRENT_BRANCH" | grep -oE "epic-[A-Z]{2,4}-[0-9]{4}" || echo "")
                if [[ -n "$EPIC_INFO" ]]; then
                    MATCHING_PRS=$(find docs/prs -name "*${EPIC_INFO}*.md" -type f 2>/dev/null | wc -l)
                    if [ $MATCHING_PRS -gt 0 ]; then
                        report_success "Found PR documentation for epic: $EPIC_INFO"
                    else
                        report_warning "No PR documentation for current branch or epic"
                    fi
                else
                    report_warning "No PR documentation for current branch"
                fi
            fi
        fi
    else
        report_warning "docs/prs/ directory exists but is empty"
    fi
else
    echo -e "${YELLOW}ℹ️  docs/prs/ directory not found (PR documentation optional)${NC}"
fi

# ============================================================================
# CHECK 6: Verify git hooks installation
# ============================================================================
echo -e "\n${BLUE}Checking git hooks installation...${NC}"

if [ -f ".git/hooks/pre-push" ]; then
    if [ -x ".git/hooks/pre-push" ]; then
        report_success "Pre-push hook installed and executable"
    else
        report_warning "Pre-push hook exists but is not executable"
        echo -e "${YELLOW}   Fix with: chmod +x .git/hooks/pre-push${NC}"
    fi
else
    report_warning "Pre-push hook not installed"
    echo -e "${YELLOW}   Install with: bash scripts/install-git-hooks.sh${NC}"
fi

# ============================================================================
# FINAL REPORT
# ============================================================================
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Validation Summary${NC}"
echo -e "${BLUE}========================================${NC}"

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}🎉 All validations passed!${NC}"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}✅ Validation completed with $WARNINGS warnings${NC}"
    echo -e "${YELLOW}Warnings are informational and don't block commits.${NC}"
    exit 0
else
    echo -e "${RED}❌ Validation failed with $ERRORS errors and $WARNINGS warnings${NC}"
    echo -e "${RED}Please fix the errors before committing changes.${NC}"
    exit 1
fi
