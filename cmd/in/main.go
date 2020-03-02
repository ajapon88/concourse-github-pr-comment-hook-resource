package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ajapon88/concourse-github-pr-comment-hook-resource"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type Request struct {
	Source  resource.Source  `json:"source"`
	Version resource.Version `json:"version"`
	Params  Params           `json:"params"`
}

type Response struct {
	Version  resource.Version  `json:"version"`
	Metadata resource.Metadata `json:"metadata"`
}

type Params struct {
	Depth int `json:"depth"`
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

	dest := os.Args[1]

	// 出力を全て標準エラーに出力する
	stdout := os.Stdout
	os.Stdout = os.Stderr

	if err := request.Source.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to validate source: %s\n", err.Error())
		os.Exit(1)
		return
	}

	client, err := resource.CreateGithubClient(&request.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create github client: %s\n", err.Error())
		os.Exit(1)
		return
	}

	// TODO: ssh
	auth := http.BasicAuth{
		Username: "x-oauth-basic",
		Password: request.Source.AccessToken,
	}

	prNumber, err := request.Version.GetPR()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	pull, err := client.GetPullRequest(prNumber)

	repository, err := git.PlainOpen(dest)
	if err != nil {
		gitURL := pull.GetHead().GetRepo().GetSVNURL()
		fmt.Fprintf(os.Stderr, "git clone %s\n", gitURL)
		repository, err = git.PlainClone(dest, false, &git.CloneOptions{
			URL:          gitURL,
			Auth:         &auth,
			SingleBranch: true,
			Depth:        request.Params.Depth,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to clone repository: %s\n", err.Error())
			os.Exit(1)
			return
		}
	}

	worktree, err := repository.Worktree()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get worktree: %s\n", err.Error())
		os.Exit(1)
		return
	}

	// fetch
	fmt.Fprintf(os.Stderr, "git fetch\n")
	err = repository.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/pull/*:refs/remotes/origin/pr/*"),
		},
		Depth: request.Params.Depth,
		Auth:  &auth,
		Tags:  git.AllTags,
	})
	if err != nil && err.Error() != "already up-to-date" {
		fmt.Fprintf(os.Stderr, "failed to fetch: %s\n", err.Error())
		os.Exit(1)
		return
	}
	// change current branch
	headBranch := fmt.Sprintf("refs/heads/%s", pull.GetHead().GetRef())
	refName := plumbing.ReferenceName(headBranch)
	ref := plumbing.NewHashReference(refName, plumbing.NewHash(request.Version.Commit))
	fmt.Fprintf(os.Stderr, "git change branch %s\n", ref)
	err = repository.Storer.SetReference(ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create branch: %s\n", err.Error())
		os.Exit(1)
		return
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: refName,
		Force:  true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to checkout: %s\n", err.Error())
		os.Exit(1)
		return
	}

	// export metadata
	metadata := resource.Metadata{
		&resource.MetadataField{Name: "pr", Value: strconv.Itoa(pull.GetNumber())},
		&resource.MetadataField{Name: "url", Value: pull.GetHTMLURL()},
		&resource.MetadataField{Name: "head_name", Value: pull.GetHead().GetRef()},
		&resource.MetadataField{Name: "head_sha", Value: request.Version.Commit},
		&resource.MetadataField{Name: "base_name", Value: pull.GetBase().GetRef()},
		&resource.MetadataField{Name: "base_sha", Value: pull.GetBase().GetSHA()},
		&resource.MetadataField{Name: "comment", Value: request.Version.Comment},
	}

	resourceDir := filepath.Join(dest, ".git", "resource")

	if f, err := os.Stat(resourceDir); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir(resourceDir, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create metadata directory: %s\n", err.Error())
			os.Exit(1)
			return
		}
	}
	if err = saveJSON(filepath.Join(resourceDir, "version.json"), request.Version); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save version file: %s\n", err.Error())
		os.Exit(1)
		return
	}
	if err = saveJSON(filepath.Join(resourceDir, "metadata.json"), metadata); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save metadata file: %s\n", err.Error())
		os.Exit(1)
		return
	}
	for _, meta := range metadata {
		if err := ioutil.WriteFile(filepath.Join(resourceDir, meta.Name), []byte(meta.Value), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write metadata file: %s\n", err.Error())
			os.Exit(1)
			return
		}
	}

	os.Stdout = stdout
	response := Response{
		Version:  request.Version,
		Metadata: metadata,
	}
	json.NewEncoder(os.Stdout).Encode(response)
}

func saveJSON(path string, v interface{}) error {
	bin, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %s", err.Error())
	}
	if err := ioutil.WriteFile(path, []byte(bin), 0644); err != nil {
		return fmt.Errorf("failed to write json file: %s", err.Error())
	}
	return nil
}
