# Git Pusher

A Go utility that monitors Git repositories specified in a YAML configuration file, checks for outgoing commits, and automatically pushes them to the remote server. It also warns about any uncommitted changes in the repositories.

## Features

- Reads repository paths from a YAML configuration file
- Checks if directories are valid Git repositories
- Detects uncommitted changes and displays warnings
- Automatically pushes outgoing commits to remote repositories
- Provides detailed status output for each repository
- Supports dry-run mode to preview changes without pushing
- Visual feedback with emojis and clear separators between repositories

## Requirements

- Go 1.21 or higher
- Git installed and configured on your system
- Valid Git repositories with remote tracking branches set up

## Configuration

Create a `config.yaml` file in the same directory as the executable with the following structure:

```yaml
repositories:
  - path: "path/to/your/repo1"
  - path: "path/to/your/repo2"
  - path: "path/to/your/repo3"
```

## Usage

1. Update the `config.yaml` file with your repository paths
2. Run the program:

```bash
# Normal mode - will push changes
go run main.go

# Dry-run mode - shows what would be pushed without actually pushing
go run main.go --dry-run
```

Or build and run the executable:

```bash
go build
# Normal mode
./git-pusher
# Dry-run mode
./git-pusher --dry-run
```

## Output

The program provides detailed output for each repository with visual indicators:

- 🔄 Repository processing status
- ⚠️ Warnings about uncommitted changes
- 📤 Information about outgoing commits
- ✅ Push operation results
- ❌ Error messages
- ℹ️ Informational messages
- 🔍 Dry-run mode indicators

Example output:
```
🔄 Processing repository: /path/to/repo1
📤 Found outgoing commits in: /path/to/repo1
✅ Successfully pushed changes from: /path/to/repo1

================================================================================

🔄 Processing repository: /path/to/repo2
⚠️ Warning: Uncommitted changes in repository: /path/to/repo2
ℹ️ No outgoing commits in: /path/to/repo2
```

## Dry-Run Mode

The `--dry-run` flag allows you to preview what changes would be pushed without actually pushing them. In this mode:

- The program shows which repositories have outgoing commits
- Displays the list of commits that would be pushed
- No actual push operations are performed
- All other checks (uncommitted changes, repository validity) are still performed

Example dry-run output:
```
🔍 Running in dry-run mode - no changes will be pushed

🔄 Processing repository: /path/to/repo1
📤 Found outgoing commits in: /path/to/repo1
🔍 [DRY-RUN] Would push changes from: /path/to/repo1
📝 Commits that would be pushed:
abc1234 Update README
def5678 Fix bug in main function
```

## Error Handling

The program handles various error conditions:
- Invalid configuration file
- Non-existent directories
- Non-Git directories
- Git command execution errors 