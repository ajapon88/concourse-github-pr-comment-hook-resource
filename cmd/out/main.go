package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ajapon88/concourse-github-pr-comment-hook-resource"
)

type Request struct {
	Source resource.Source `json:"source"`
	Params Params          `json:"params"`
}

type Params struct {
	Path        string `json:"path"`
	BaseContext string `json:"base_context"`
	Context     string `json:"context"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type Response struct {
	Version  resource.Version  `json:"version"`
	Metadata resource.Metadata `json:"metadata"`
}

func main() {
	var request Request

	infoEncoder := json.NewEncoder(os.Stderr)
	infoEncoder.SetIndent("", "    ")

	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode: %s\n", err.Error())
		os.Exit(1)
		return
	}

	src := os.Args[1]

	if err := request.Source.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to validate source: %s\n", err.Error())
		os.Exit(1)
		return
	}

	if err := request.Params.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
		return
	}

	resourceDir := filepath.Join(src, request.Params.Path, ".git", "resource")

	var version resource.Version
	if err := loadJSON(filepath.Join(resourceDir, "version.json"), &version); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	fmt.Fprintf(os.Stderr, "version:\n")
	infoEncoder.Encode(version)
	var metadata resource.Metadata
	if err := loadJSON(filepath.Join(resourceDir, "metadata.json"), &metadata); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	fmt.Fprintf(os.Stderr, "metadata:\n")
	infoEncoder.Encode(metadata)

	client, err := resource.CreateGithubClient(&request.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create github client: %s\n", err.Error())
		os.Exit(1)
		return
	}
	repoStatus, err := client.UpdateCommitStatus(version.Commit, request.Params.Status, request.Params.TargetURL, request.Params.Description, request.Params.BaseContext, request.Params.Context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to update commit status: %s\n", err.Error())
		os.Exit(1)
		return
	}
	fmt.Fprintf(os.Stderr, "RepoStatus:\n")
	infoEncoder.Encode(repoStatus)

	response := Response{
		version,
		metadata,
	}

	json.NewEncoder(os.Stdout).Encode(response)
}

func (params *Params) Validate() error {
	// check status
	allow := false
	for _, status := range []string{"error", "failure", "pending", "success"} {
		if status == params.Status {
			allow = true
			break
		}
	}
	if !allow {
		return fmt.Errorf("invalid status")
	}
	return nil
}

func loadJSON(path string, v interface{}) error {
	bin, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read json file: %s", err.Error())
	}

	if err := json.Unmarshal(bin, &v); err != nil {
		return fmt.Errorf("failed to marshal json json: %s", err.Error())
	}
	return nil
}
