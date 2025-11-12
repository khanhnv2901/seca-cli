# SECA-CLI Documentation

Complete documentation for the Secure Engagement & Compliance Auditing CLI.

## Table of Contents

### 📖 User Guides

Essential documentation for getting started and daily usage.

- **[Installation Guide](user-guide/installation.md)** - Installation instructions for all platforms
- **[Configuration Guide](user-guide/configuration.md)** - Complete configuration reference

### 👤 Operator Guides

Documentation for security operators conducting authorized testing.

- **[Operator Training Guide](operator-guide/operator-training.md)** - Training materials and certification
- **[Compliance Guide](operator-guide/compliance.md)** - Compliance requirements and best practices

### 🔧 Technical Documentation

In-depth technical information for developers and system administrators.

- **[Deployment Guide](technical/deployment.md)** - Production deployment instructions
- **[Testing Guide](technical/testing.md)** - Testing and quality assurance
- **[Version Management Guide](technical/version-management.md)** - Build versioning and releases

### 📚 Reference

Additional reference materials and guides.

- **[Data Migration Guide](reference/data-migration.md)** - Migrating data directories
- **[Template Approaches](reference/template-approaches.md)** - Report template implementation

---

## Quick Links

### Essential Reading

For new users, start here:

1. [Installation Guide](user-guide/installation.md) - Get SECA-CLI installed
2. [Operator Training Guide](operator-guide/operator-training.md) - Learn the fundamentals
3. [Compliance Guide](operator-guide/compliance.md) - Understand compliance requirements
4. [Configuration Guide](user-guide/configuration.md) - Configure your environment

### Common Tasks

- **Install SECA-CLI**: [Installation Guide](user-guide/installation.md#quick-install)
- **Configure Settings**: [Configuration Guide](user-guide/configuration.md)
- **First Engagement**: [Operator Training Guide](operator-guide/operator-training.md#hands-on-exercises)
- **Deploy to Production**: [Deployment Guide](technical/deployment.md)
- **Run Tests**: [Testing Guide](technical/testing.md)
- **Build Releases**: [Version Management Guide](technical/version-management.md)
- **Migrate Data**: [Data Migration Guide](reference/data-migration.md)

### By Audience

#### 🆕 New Users
- [Installation Guide](user-guide/installation.md)
- [Operator Training Guide](operator-guide/operator-training.md)
- [Configuration Guide](user-guide/configuration.md)

#### 🛡️ Security Operators
- [Operator Training Guide](operator-guide/operator-training.md)
- [Compliance Guide](operator-guide/compliance.md)
- [Configuration Guide](user-guide/configuration.md)

#### 👨‍💼 Compliance Officers
- [Compliance Guide](operator-guide/compliance.md)
- [Data Migration Guide](reference/data-migration.md)
- [Deployment Guide](technical/deployment.md)

#### 👨‍💻 Developers
- [Testing Guide](technical/testing.md)
- [Version Management Guide](technical/version-management.md)
- [Template Approaches](reference/template-approaches.md)

#### 🏢 System Administrators
- [Installation Guide](user-guide/installation.md)
- [Deployment Guide](technical/deployment.md)
- [Configuration Guide](user-guide/configuration.md)

---

## Documentation Structure

```
docs/
├── README.md (this file)
│
├── user-guide/
│   ├── installation.md         Installation for all platforms
│   └── configuration.md        Configuration reference
│
├── operator-guide/
│   ├── operator-training.md    Operator training materials
│   └── compliance.md           Compliance requirements
│
├── technical/
│   ├── deployment.md           Production deployment
│   ├── testing.md              Testing documentation
│   └── version-management.md   Build and release management
│
└── reference/
    ├── data-migration.md       Data directory migration
    └── template-approaches.md  Report template details
```

---

## External Resources

### Main Project Resources

- **[Main README](../README.md)** - Project overview and quick start
- **[CHANGELOG](../CHANGELOG.md)** - Version history and release notes
- **[Example Config](../.seca-cli.yaml.example)** - Configuration file example

### Online Resources

- **Repository**: https://github.com/khanhnv2901/seca-cli
- **Issues**: https://github.com/khanhnv2901/seca-cli/issues
- **Releases**: https://github.com/khanhnv2901/seca-cli/releases

### Getting Help

- **GitHub Issues**: Report bugs or request features
- **Email**: khanhnv2901@gmail.com

---

## Documentation Standards

### For Contributors

When adding or updating documentation:

1. **Follow the structure**: Place docs in appropriate category folders
2. **Use relative links**: Link to other docs using relative paths
3. **Update this index**: Add new documents to the appropriate section
4. **Keep it current**: Update docs when code changes
5. **Be comprehensive**: Include examples and troubleshooting

### File Naming

- Use lowercase with hyphens: `file-name.md`
- Be descriptive: `operator-training.md` not `training.md`
- Avoid special characters

### Link Format

```markdown
<!-- Good: Relative links with descriptive text -->
[Configuration Guide](user-guide/configuration.md)

<!-- Bad: Absolute links or bare URLs -->
[Configuration](../docs/user-guide/configuration.md)
```

---

## Contributing to Documentation

Documentation improvements are always welcome! To contribute:

1. Fork the repository
2. Create a feature branch: `git checkout -b docs/improve-config-guide`
3. Make your changes
4. Update this index if adding new docs
5. Test all links
6. Submit a pull request

See the [main CONTRIBUTING guide](../README.md#contributing) for more details.

---

## License

All documentation is licensed under the MIT License, same as the project code.

---

**Last Updated**: November 2025
**SECA-CLI Version**: 1.2.0+

For the latest documentation, always refer to the repository: https://github.com/khanhnv2901/seca-cli
