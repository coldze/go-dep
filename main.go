package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type GovendorPackage struct {
	CheckSum     string `json:"checksumSHA1,omitempty"`
	Path         string `json:"path"`
	Revisiong    string `json:"revision"`
	RevisionTime string `json:"revisionTime,omitempty"`
	Version      string `json:"version.omitempty"`
	VersionExact string `json:"versionExact,omitempty"`
}

type GovendorConfig struct {
	Packages []*GovendorPackage `json:"package"`
}

const (
	BRANCH_DEVELOP = "develop"
	BRANCH_STAGE   = "stage"
	BRANCH_MASTER  = "master"
)

func composePath(root string, subdir string) string {
	return strings.TrimRight(root, "/") + "/" + strings.TrimLeft(subdir, "/")
}

func parseVendor(root string) (*GovendorConfig, error) {
	data, err := ioutil.ReadFile(composePath(root, "vendor/vendor.json"))
	if err != nil {
		return nil, err
	}
	config := &GovendorConfig{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func fixPackage(pckg *GovendorPackage, branches []string) error {
	for i := range branches {
		err := exec.Command("govendor", "fetch", fmt.Sprintf("%v@%v", pckg.Path, branches[i])).Run()
		if err != nil {
			log.Printf("Failed to switch '%v' to branch '%v'. Error: %v", pckg.Path, branches[i], err)
			continue
		}
		log.Printf("Switched '%v' to branch '%v'", pckg.Path, branches[i])
		return nil
	}
	return nil
}

func removeOldSources(path string, pckg *GovendorPackage) error {
	return exec.Command("rm", "-rf", composePath(composePath(path, "vendor"), pckg.Path)).Run()
}

func prepareBranches(branch string) []string {
	branches := []string{BRANCH_MASTER}
	if branch == BRANCH_MASTER {
		return branches
	}
	branches = append([]string{BRANCH_STAGE}, branches...)
	if branch == BRANCH_STAGE {
		return branches
	}
	branches = append([]string{BRANCH_DEVELOP}, branches...)
	if branch == BRANCH_DEVELOP {
		return branches
	}
	return append([]string{branch}, branches...)
}

func processConfig(config *GovendorConfig, repo string, branch string, path string) error {
	branches := prepareBranches(branch)
	repoLower := strings.ToLower(repo)
	log.Printf("Got packages: %v", len(config.Packages))
	for i := range config.Packages {
		pckg := config.Packages[i]
		if strings.Contains(strings.ToLower(pckg.Path), repoLower) {
			log.Printf("[%v] Running fix-package for: %v", i, pckg.Path)
			err := fixPackage(pckg, branches)
			if err != nil {
				log.Printf("[%v] Running fix-package failed", i)
				continue
			}
			log.Printf("[%v] Running fix-package done", i)
			removeOldSources(path, pckg)
			log.Printf("[%v] Removed old", i)
		} else {
			log.Printf("[%v] Nothing to do.", i)
		}
	}
	log.Printf("Processing done")
	return nil
}

func syncSources() error {
	return exec.Command("govendor", "sync").Run()
}

func runMain() {

}

func main() {
	workingDir := flag.String("path", "", "working-dir (full path)")
	branch := flag.String("branch", "master", "source branch")
	repoPath := flag.String("rep", "github.com/coldze", "Repo path")
	flag.Parse()
	if (workingDir == nil) || (len(*workingDir) <= 0) {
		log.Printf("Working dir not specified")
		os.Exit(1)
	}
	if (branch == nil) || len(*branch) <= 0 {
		log.Printf("Branch not specified")
		os.Exit(1)
	}

	err := os.Chdir(*workingDir)
	if err != nil {
		log.Printf("Failed to change dir to '%v'. Error: %v", *workingDir, err)
		os.Exit(1)
	}

	if (repoPath == nil) || len(*repoPath) <= 0 {
		log.Printf("Repository not specified")
		os.Exit(1)
	}

	config, err := parseVendor(*workingDir)
	if err != nil {
		log.Printf("Failed to parse vendor. Directory: %v. Error: %v", *workingDir, err)
		os.Exit(1)
	}

	err = processConfig(config, *repoPath, *branch, *workingDir)
	if err != nil {
		log.Printf("Failed to process config. Repo: %v. Branch: %v. Dir: %v. Error: %v", *repoPath, *branch, *workingDir, err)
		os.Exit(1)
	}
	err = syncSources()
	if err != nil {
		log.Printf("Failed to sync sources. Error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}
