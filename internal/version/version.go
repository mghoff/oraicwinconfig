package version

import "fmt"

var (
	Version = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GoVersion = "unknown"
)

func GetVersionInfo() string {
	return fmt.Sprintf(
		"Version: %s\nBuild Time: %s\nGit Commit: %s\nGo Version: %s\n",
		Version, BuildTime, GitCommit, GoVersion)
}