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
	Path            string `json:"path"`
	BaseContext     string `json:"base_context"`
	Context         string `json:"context"`
	TargetURL       string `json:"target_url"`
	Description     string `json:"description"`
	DescriptionFile string `json:"description_file"`
	Comment         string `json:"comment"`
	CommentFile     string `json:"comment_file"`
	Status          string `json:"status"`
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}

	resourceDir := filepath.Join(src, request.Params.Path, ".git", "resource")

	var version resource.Version
	if err := loadJSON(filepath.Join(resourceDir, "version.json"), &version); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	fmt.Fprintf(os.Stderr, "version:\n")
	infoEncoder.Encode(version)
	var metadata resource.Metadata
	if err := loadJSON(filepath.Join(resourceDir, "metadata.json"), &metadata); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	fmt.Fprintf(os.Stderr, "metadata:\n")
	infoEncoder.Encode(metadata)

	prNumber, err := version.GetPR()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}

	description, err := request.Params.GetDescription(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	comment, err := request.Params.GetComment(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	client, err := resource.CreateGithubClient(&request.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create github client: %s\n", err.Error())
		os.Exit(1)
		return
	}

	fmt.Fprintf(os.Stderr, "update commit status: '%s'\n", request.Params.Status)
	repoStatus, err := client.UpdateCommitStatus(version.Commit, request.Params.Status, request.Params.TargetURL, description, request.Params.BaseContext, request.Params.Context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to update commit status: %s\n", err.Error())
		os.Exit(1)
		return
	}
	fmt.Fprintf(os.Stderr, "RepoStatus:\n")
	infoEncoder.Encode(repoStatus)

	if comment != "" {
		fmt.Fprintf(os.Stderr, "post comment: \"%s\"\n", comment)
		issueComment, err := client.PostComment(prNumber, comment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to post comment: %s\n", err.Error())
			os.Exit(1)
			return
		}
		fmt.Fprintf(os.Stderr, "IssueComment:\n")
		infoEncoder.Encode(issueComment)
	}

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

func (params *Params) GetDescription(src string) (string, error) {
	if params.DescriptionFile != "" {
		description, err := ioutil.ReadFile(filepath.Join(src, params.DescriptionFile))
		if err != nil {
			return "", fmt.Errorf("failed to read description file '%s' : %s", params.DescriptionFile, err.Error())
		}
		return string(description), nil
	}

	return params.Description, nil
}

func (params *Params) GetComment(src string) (string, error) {
	if params.CommentFile != "" {
		comment, err := ioutil.ReadFile(filepath.Join(src, params.CommentFile))
		if err != nil {
			return "", fmt.Errorf("failed to read comment file '%s' : %s", params.CommentFile, err.Error())
		}
		return string(comment), nil
	}

	return params.Comment, nil
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
