# Pull Request

## Description

<!-- Provide a brief description of the changes in this PR -->

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Code refactoring
- [ ] Performance improvement
- [ ] Test update

## Related Issues

<!-- Link to related issues, e.g., "Fixes #123" or "Relates to #456" -->

## Testing

### Unit Tests
- [ ] All existing unit tests pass
- [ ] New unit tests added for new functionality
- [ ] Unit tests cover edge cases and error conditions
- [ ] Coverage has not decreased (run `make test-coverage`)

### Integration Tests
- [ ] All integration tests pass
- [ ] Integration tests updated if workflow changed
- [ ] Manual testing performed

### Test Commands Run
```bash
# Paste output of test commands here
make test
make test-integration
```

## Checklist

### Code Quality
- [ ] Code follows the project's style guidelines
- [ ] Self-review of code performed
- [ ] Code is well-commented, particularly in hard-to-understand areas
- [ ] No unnecessary debug/console statements
- [ ] No security vulnerabilities introduced

### Documentation
- [ ] README.md updated (if applicable)
- [ ] COMPLIANCE.md updated (if compliance features changed)
- [ ] TESTING.md updated (if testing approach changed)
- [ ] Code comments added/updated
- [ ] API documentation updated (if applicable)

### Safety & Security
- [ ] Authorization checks are in place for new commands
- [ ] No hardcoded credentials or sensitive data
- [ ] Input validation implemented
- [ ] Error messages don't leak sensitive information
- [ ] Follows principle of least privilege

### Compliance (if applicable)
- [ ] Audit trail maintained for new operations
- [ ] SHA256 hashing implemented for new evidence files
- [ ] Retention policies respected
- [ ] Operator attribution included
- [ ] ROE confirmation required for security operations

### Git
- [ ] Commits are atomic and well-described
- [ ] Commit messages follow conventional commits format
- [ ] No merge conflicts
- [ ] Branch is up to date with main/develop

## Screenshots/Output (if applicable)

<!-- Add screenshots or command output demonstrating the changes -->

```
# Paste relevant output here
```

## Performance Impact

<!-- Describe any performance implications -->

- [ ] No significant performance impact
- [ ] Performance improved
- [ ] Performance regression (explain why acceptable)

## Breaking Changes

<!-- If this PR contains breaking changes, describe them and the migration path -->

## Additional Notes

<!-- Any additional information that reviewers should know -->

---

By submitting this pull request, I confirm that:
- [ ] I have read and followed the contributing guidelines
- [ ] My code follows the security best practices outlined in the project
- [ ] I have tested my changes thoroughly
- [ ] I understand this tool is for authorized security testing only
