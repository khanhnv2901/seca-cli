# Data Directory Migration Guide

## Overview

As of version 0.2.0, SECA-CLI now stores user data in OS-appropriate data directories instead of the project directory. This improves security, follows OS standards, and enables proper multi-user support.

## New Data Locations

### Linux/Unix
```
~/.local/share/seca-cli/
├── engagements.json
└── results/
    └── <engagement-id>/
        ├── audit.csv
        ├── results.json
        └── ...
```

### macOS
```
~/Library/Application Support/seca-cli/
├── engagements.json
└── results/
    └── <engagement-id>/
        ├── audit.csv
        ├── results.json
        └── ...
```

### Windows
```
%LOCALAPPDATA%\seca-cli\
├── engagements.json
└── results\
    └── <engagement-id>\
        ├── audit.csv
        ├── results.json
        └── ...
```

## Automatic Migration

The tool automatically migrates data from the old location (project directory) to the new location on first run.

### What Happens During Migration

1. **On first run**, if `engagements.json` exists in the current directory:
   - File is copied to the new data directory
   - Original file is renamed to `engagements.json.backup`
   - You'll see a message: `Migrated engagements.json from ./engagements.json to <new-path>`

2. **Results directory** migration:
   - The `results/` directory in the project root is now deprecated
   - New results are stored in the OS-appropriate data directory
   - Old results in `./results/` are NOT automatically migrated
   - You can manually move them if needed (see below)

## Manual Migration (if needed)

If automatic migration doesn't work or you want to migrate manually:

### Step 1: Find Your Data Directory

Run this command to see where data is stored:
```bash
# The tool will print the data directory on startup
./seca engagement list 2>&1 | grep "results_dir"
```

Or check based on your OS:
- **Linux**: `~/.local/share/seca-cli/`
- **macOS**: `~/Library/Application Support/seca-cli/`
- **Windows**: `%LOCALAPPDATA%\seca-cli\`

### Step 2: Migrate Engagements

```bash
# Linux/macOS
cp engagements.json ~/.local/share/seca-cli/engagements.json

# Windows
copy engagements.json %LOCALAPPDATA%\seca-cli\engagements.json
```

### Step 3: Migrate Results (Optional)

```bash
# Linux/macOS
cp -r results/* ~/.local/share/seca-cli/results/

# Windows
xcopy /E results %LOCALAPPDATA%\seca-cli\results\
```

### Step 4: Verify Migration

```bash
# List engagements to verify they were migrated
./seca engagement list

# Check a specific engagement's results
./seca report stats --id=<engagement-id>
```

### Step 5: Clean Up (Optional)

Once you've verified the migration:
```bash
# Remove old files from project directory
rm -rf engagements.json engagements.json.backup results/
```

## Override Data Directory

You can override the default data directory using the config file:

**~/.seca-cli.yaml** (or wherever your config is):
```yaml
results_dir: "/custom/path/to/results"
```

This is useful for:
- Shared team directories
- Network storage
- Custom backup solutions

## Backward Compatibility

For backward compatibility with scripts and workflows:
- The old location still works if you set `results_dir` in config
- Tests use isolated directories and work regardless of data location
- The `resultsDir` variable can still be overridden

## Benefits of New Location

✅ **User-specific data** - No conflicts between users
✅ **Proper permissions** - User-owned directories
✅ **OS standards** - Follows XDG and platform conventions
✅ **Clean project** - Source code separate from user data
✅ **Easy backup** - All data in one predictable location
✅ **Multi-user support** - Each user has their own data

## Troubleshooting

### "Permission denied" error

The tool should automatically create directories with proper permissions (`0755`). If you get permission errors:

```bash
# Create directory manually
mkdir -p ~/.local/share/seca-cli
chmod 755 ~/.local/share/seca-cli
```

### Migration didn't happen

Check if the old file exists:
```bash
ls -la engagements.json
```

If it exists but wasn't migrated, check stderr output for migration messages.

### Data in wrong location

Set the `results_dir` explicitly in your config:
```yaml
# ~/.seca-cli.yaml
results_dir: "/path/to/your/data"
```

### Need to revert to old behavior

Edit your config file:
```yaml
# ~/.seca-cli.yaml
results_dir: "./results"  # Use project directory
```

And manually copy data back:
```bash
cp ~/.local/share/seca-cli/engagements.json ./
```

## For Developers

### Testing

Tests use isolated temporary directories and work regardless of data location:
```go
// Tests automatically handle data directory isolation
func TestSomething(t *testing.T) {
    cleanup := setupTestEngagements(t)
    defer cleanup()
    // Test code...
}
```

### API

```go
// Get OS-appropriate data directory
dataDir, err := getDataDir()

// Get engagements file path (with automatic migration)
filePath, err := getEngagementsFilePath()

// Get results directory
resultsDir, err := getResultsDir()
```

## Migration Checklist

- [ ] Backup current `engagements.json` and `results/`
- [ ] Run the tool (migration happens automatically)
- [ ] Verify engagements: `./seca engagement list`
- [ ] Verify results: `./seca report stats --id=<id>`
- [ ] Update any scripts that reference `./engagements.json`
- [ ] Update backup scripts to use new location
- [ ] Clean up old files (optional)
- [ ] Update team documentation

## Questions?

- Check data location: Tool prints it on startup
- View migration messages: Check stderr output
- Override location: Use config file
- Need help: https://github.com/khanhnv2901/seca-cli/issues
