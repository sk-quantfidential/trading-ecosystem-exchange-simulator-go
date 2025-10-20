# PR Title: {prefix}(epic-XXX-9999): Short description of change

<!--
CUSTOMIZATION INSTRUCTIONS:
1. Copy this template to: docs/prs/{branch-name}.md
2. Replace {prefix} with: feat, fix, docs, style, refac, test, chore, or ci
3. Replace XXX with your 3-letter project code (e.g., CLD, API, WEB)
4. Replace 9999 with your 4-digit epic number
5. Fill in all sections below
6. Remove placeholder text and comments before committing
7. Keep this under 1000 words - be concise but complete

Example filename: docs/prs/feature-epic-CLD-0013-create-git-standards-skill.md
Example title: feat(epic-CLD-0013): create portable git quality standards skill

DELETE THIS COMMENT BLOCK BEFORE COMMITTING
-->

## Summary

<!--
Provide a brief overview of the changes in 2-3 sentences.
Focus on WHAT changed and WHY, not HOW (code details go in Files Changed).
This should be understandable by non-technical stakeholders.

Good example:
"Implemented a new authentication flow using JWT tokens to replace the existing
session-based auth. This reduces server memory usage and enables horizontal scaling
of the API service."

Bad example:
"Added new files for auth. Updated the login endpoint. Changed some config."
-->

**What**: [Brief description of what was changed]

**Why**: [Business justification or problem being solved]

**Impact**: [Who/what is affected by this change]

## Epic/Milestone Reference

**Epic**: epic-XXX-9999
**Milestone**: [Milestone name from TODO.md]
**Related Issues**: #[issue-number], #[issue-number]
**Related PRs**: #[pr-number] (if applicable)

## Type of Change

<!-- Check all that apply -->

- [ ] New feature (non-breaking change that adds functionality)
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Code refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test coverage improvement
- [ ] CI/CD or tooling change
- [ ] Dependency update

## Files Changed

<!--
List the major files/modules changed and what changed in each.
Use this format:
- `path/to/file.ext` - Brief description of what changed and why
- `another/file.ext` - What changed

For large PRs (>10 files), group by category:
### Core Logic
- Files...

### Tests
- Files...

### Documentation
- Files...
-->

### Core Changes

- `path/to/main/file.ext` - [Description of changes]
- `path/to/another/file.ext` - [Description of changes]

### Tests

- `path/to/test/file.ext` - [Description of test coverage]

### Documentation

- `path/to/docs/file.md` - [Description of doc updates]

### Configuration

- `path/to/config/file.yml` - [Description of config changes]

## Quality Assurance

### Testing Completed

<!--
Describe what testing you performed.
Be specific - what scenarios did you test? What edge cases?
Include commands run and results observed.
-->

- [ ] **Unit Tests**: [What was tested, results]
  ```bash
  # Command run
  npm test

  # Result
  All 47 tests passed
  ```

- [ ] **Integration Tests**: [What was tested, results]
  ```bash
  # Command run
  npm run test:integration

  # Result
  All integration tests passed
  ```

- [ ] **Manual Testing**: [What scenarios were tested]
  - Scenario 1: [Description and result]
  - Scenario 2: [Description and result]
  - Edge case: [Description and result]

- [ ] **Performance Testing**: [If applicable]
  - Baseline: [Performance before changes]
  - After changes: [Performance after changes]
  - Impact: [Improvement or regression details]

### Test Coverage

- **New Code Coverage**: [Percentage, e.g., 95%]
- **Overall Coverage Change**: [Percentage change, e.g., +2.3%]
- **Uncovered Areas**: [List any code not covered and why]

### Validation Checks Passed

- [ ] Markdown linting (if applicable)
- [ ] Code linting/formatting
- [ ] Type checking (if applicable)
- [ ] Build successful
- [ ] Pre-push hooks passed
- [ ] All CI checks passed

## Breaking Changes

<!--
If this is a breaking change, describe:
1. What breaks
2. How to migrate
3. Deprecation timeline (if applicable)

If not a breaking change, write "None"
-->

**Breaking**: [Yes/No]

**Details**: [What breaks and why]

**Migration Guide**:
```bash
# Steps to update for this change
# 1. ...
# 2. ...
```

**Deprecation Timeline**: [When old behavior will be removed, if applicable]

## Deployment Notes

<!--
Special instructions for deploying this change.
Include database migrations, config changes, feature flags, etc.

If no special deployment needs, write "Standard deployment process"
-->

### Pre-Deployment

- [ ] [Action required before deploying, e.g., "Update environment variable X"]
- [ ] [Database migration needed: describe]
- [ ] [Config file update needed: describe]

### Post-Deployment

- [ ] [Action required after deploying, e.g., "Restart service Y"]
- [ ] [Verification steps to confirm deployment success]
- [ ] [Rollback plan if deployment fails]

### Feature Flags

- [ ] This change is behind feature flag: `[flag-name]`
- [ ] Feature flag enabled in: [environments]

## Security Considerations

<!--
Describe any security implications:
- New authentication/authorization logic
- Changes to data access patterns
- New external dependencies
- Exposure of new endpoints/APIs
- Changes to sensitive data handling

If no security implications, write "No security impact"
-->

**Security Impact**: [None/Low/Medium/High]

**Details**:
- [Security consideration 1]
- [Security consideration 2]

**Mitigations**:
- [How security concerns are addressed]

## Dependencies

<!--
List new dependencies added or updated.
Include version numbers and justification.

If no dependency changes, write "No new dependencies"
-->

### Added

- `package-name@version` - [Why this dependency was added]

### Updated

- `package-name` from `old-version` to `new-version` - [Why updated, breaking changes]

### Removed

- `package-name` - [Why removed, what replaced it]

## Rollback Plan

<!--
How to rollback if this change causes issues in production.
Be specific - what commands, what steps.
-->

**Rollback Steps**:
1. [Step to revert deployment]
2. [Step to restore previous state]
3. [Step to verify rollback successful]

**Rollback Risk**: [Low/Medium/High]
**Rollback Time**: [Estimated time to complete rollback]

## Screenshots/Videos

<!--
For UI changes, include before/after screenshots or videos.
For API changes, include example requests/responses.
For CLI changes, include terminal output.

Delete this section if not applicable.
-->

### Before

![Before](url-to-image)

### After

![After](url-to-image)

## Documentation Updates

<!--
List what documentation was updated.
Include README, API docs, architecture docs, runbooks, etc.
-->

- [ ] README.md updated
- [ ] API documentation updated
- [ ] Architecture documentation updated
- [ ] Runbook/operational docs updated
- [ ] TODO.md updated with completed tasks
- [ ] Changelog updated (if applicable)

## Reviewer Notes

<!--
Special instructions for reviewers.
What should they focus on? What areas need extra scrutiny?
Are there specific concerns you want feedback on?
-->

**Focus Areas**:
- [Area of code that needs careful review]
- [Design decision you want feedback on]

**Known Issues**:
- [Issue you're aware of but haven't addressed yet]

**Follow-up Work**:
- [Work that will be done in a future PR]

## Checklist

<!--
Final checklist before submitting PR.
All items should be checked before requesting review.
-->

- [ ] Code follows project style guidelines
- [ ] Self-review of code completed
- [ ] Code commented in hard-to-understand areas
- [ ] Documentation updated to reflect changes
- [ ] No new warnings generated
- [ ] Tests added for new functionality
- [ ] All tests passing locally
- [ ] Dependent changes merged and published
- [ ] PR title follows convention: `{prefix}(epic-XXX-9999): description`
- [ ] Branch name follows convention: `type/epic-XXX-9999-milestone-behavior`
- [ ] TODO.md updated with task completion
- [ ] No merge conflicts with target branch

## Additional Context

<!--
Any other context, background, or information reviewers should know.
Links to design docs, discussions, research, related work.

Delete this section if not needed.
-->

**References**:
- [Link to design document]
- [Link to discussion/RFC]
- [Link to research/investigation]

**Alternative Approaches Considered**:
- [Approach 1 and why it wasn't chosen]
- [Approach 2 and why it wasn't chosen]

---

<!--
Template Version: 1.0.0
Skill: git_quality_standards
Last Updated: 2025-01-20

DELETE ALL COMMENT BLOCKS AND PLACEHOLDER TEXT BEFORE COMMITTING
-->
