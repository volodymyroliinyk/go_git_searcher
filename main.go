package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type GitProject struct {
	Path           string
	ProjectName    string
	RemoteRepo     string
	LastCommitDate time.Time
}

func main() {
	directoryFlag := flag.String("directory", "", "Path to the root directory to search for Git repositories")
	flag.Parse()

	if *directoryFlag == "" {
		fmt.Println("❌ Please provide --directory=/path/to/root")
		os.Exit(1)
	}

	rootDir := *directoryFlag
	var gitProjects []GitProject

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			projectPath := filepath.Dir(path)
			projectName := filepath.Base(projectPath)
			remoteRepo, lastCommitDate, err := getGitInfo(projectPath)

			if err != nil {
				fmt.Printf("❌ [%s] %v\n", projectPath, err)
				return nil
			}

			gitProjects = append(gitProjects, GitProject{
				Path:           projectPath,
				ProjectName:    projectName,
				RemoteRepo:     remoteRepo,
				LastCommitDate: lastCommitDate,
			})
		}

		return nil
	})

	if err != nil {
		fmt.Printf("🚫 Error while scanning directory: %v\n", err)
		return
	}

	// Sorting
	sort.Slice(gitProjects, func(i, j int) bool {
		if gitProjects[i].RemoteRepo != "" && gitProjects[j].RemoteRepo != "" {
			if gitProjects[i].RemoteRepo == gitProjects[j].RemoteRepo {
				return gitProjects[i].LastCommitDate.After(gitProjects[j].LastCommitDate)
			}

			return gitProjects[i].RemoteRepo < gitProjects[j].RemoteRepo
		}

		return gitProjects[i].ProjectName < gitProjects[j].ProjectName
	})

	// Create CSV
	csvFile, err := os.Create("git_projects_report.csv")
	if err != nil {
		fmt.Printf("❌ Failed to create CSV file: %v\n", err)

		return
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	writer.Write([]string{"Project name", "Path", "Remote repository", "Last commit date"})

	for _, project := range gitProjects {
		writer.Write([]string{
			project.ProjectName,
			project.Path,
			project.RemoteRepo,
			project.LastCommitDate.Format("2006-01-02 15:04:05"),
		})
	}

	fmt.Println("✅ Report saved to 'git_projects_report.csv'")
}

func getGitInfo(projectPath string) (string, time.Time, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectPath
	remoteRepoBytes, err := cmd.Output()
	remoteRepo := ""

	if err == nil {
		remoteRepo = strings.TrimSpace(string(remoteRepoBytes))
	}

	cmd = exec.Command("git", "log", "-1", "--format=%cd", "--date=iso")
	cmd.Dir = projectPath

	lastCommitBytes, err := cmd.Output()
	if err != nil {
		return remoteRepo, time.Time{}, fmt.Errorf("Failed to get last commit date")
	}

	lastCommitDateStr := strings.TrimSpace(string(lastCommitBytes))
	lastCommitDate, err := time.Parse("2006-01-02 15:04:05 -0700", lastCommitDateStr)
	if err != nil {
		return remoteRepo, time.Time{}, fmt.Errorf("Failed to parse commit date: %v", err)
	}

	return remoteRepo, lastCommitDate, nil
}
