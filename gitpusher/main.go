package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repositories []Repository `yaml:"repositories"`
}

type Repository struct {
	Path string `yaml:"path"`
}

func main() {
	// Parse command line flags
	dryRun := flag.Bool("dry-run", false, "Show what would be done without actually pushing changes")
	flag.Parse()

	// Read config file
	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("❌ Error reading config file: %v\n", err)
		os.Exit(1)
	}

	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		fmt.Printf("❌ Error parsing config file: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("🔍 Running in dry-run mode - no changes will be pushed")
	}

	// Process each repository
	for i, repo := range config.Repositories {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("=", 80))
		}
		fmt.Printf("\n🔄 Processing repository: %s\n", repo.Path)

		// Check if directory exists
		if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
			fmt.Printf("❌ Directory does not exist: %s\n", repo.Path)
			continue
		}

		// Check if it's a git repository
		if !isGitRepository(repo.Path) {
			fmt.Printf("❌ Not a git repository: %s\n", repo.Path)
			continue
		}

		// Check for uncommitted changes
		if hasUncommittedChanges(repo.Path) {
			fmt.Printf("⚠️ Warning: Uncommitted changes in repository: %s\n", repo.Path)
		}

		// Check for outgoing commits and push if any
		if hasOutgoingCommits(repo.Path) {
			fmt.Printf("📤 Found outgoing commits in: %s\n", repo.Path)
			if *dryRun {
				fmt.Printf("🔍 [DRY-RUN] Would push changes from: %s\n", repo.Path)
				// Show what would be pushed
				showOutgoingCommits(repo.Path)
			} else {
				if err := pushRepository(repo.Path); err != nil {
					fmt.Printf("❌ Error pushing repository: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully pushed changes from: %s\n", repo.Path)
				}
			}
		} else {
			fmt.Printf("ℹ️ No outgoing commits in: %s\n", repo.Path)
		}
	}
}

func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return !os.IsNotExist(err)
}

func hasUncommittedChanges(path string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Error checking git status: %v\n", err)
		return false
	}
	return len(output) > 0
}

func hasOutgoingCommits(path string) bool {
	cmd := exec.Command("git", "log", "@{push}..")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		// If there's an error, it might mean no upstream branch is set
		return false
	}
	return len(output) > 0
}

func showOutgoingCommits(path string) {
	cmd := exec.Command("git", "log", "--oneline", "@{push}..")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Error showing outgoing commits: %v\n", err)
		return
	}
	if len(output) > 0 {
		fmt.Println("📝 Commits that would be pushed:")
		fmt.Print(string(output))
	}
}

func pushRepository(path string) error {
	cmd := exec.Command("git", "push")
	cmd.Dir = path
	return cmd.Run()
}
