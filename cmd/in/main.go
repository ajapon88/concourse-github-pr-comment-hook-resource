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
	Version  resource.Version `json:"version"`
	Metadata Metadata         `json:"metadata"`
}

type Params struct {
	Depth int `json:"depth"`
}

type Metadata []*resource.MetadataField

func main() {
	var request Request
	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode: %s\n", err.Error())
		os.Exit(1)
		return
	}

	dest := os.Args[1]
	fmt.Fprintf(os.Stderr, "version: %v\n", request.Version)
	fmt.Fprintf(os.Stderr, "source: %v\n", request.Source)
	fmt.Fprintf(os.Stderr, "params: %v\n", request.Params)

	// 出力を全て標準エラーに出力する
	stdout := os.Stdout
	os.Stdout = os.Stderr

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

	prNumber, err := strconv.Atoi(request.Version.PR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to atoi pr number: %s\n", err.Error())
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
	ref := plumbing.NewHashReference(refName, plumbing.NewHash(pull.GetHead().GetSHA()))
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
	metadata := Metadata{
		&resource.MetadataField{Name: "pr", Value: strconv.Itoa(pull.GetNumber())},
		&resource.MetadataField{Name: "url", Value: pull.GetHTMLURL()},
		&resource.MetadataField{Name: "head_name", Value: pull.GetHead().GetRef()},
		&resource.MetadataField{Name: "head_sha", Value: pull.GetHead().GetSHA()},
		&resource.MetadataField{Name: "base_name", Value: pull.GetBase().GetRef()},
		&resource.MetadataField{Name: "base_sha", Value: pull.GetBase().GetSHA()},
		&resource.MetadataField{Name: "comment", Value: request.Version.Comment},
	}

	metaDir := filepath.Join(dest, ".git", "resource")

	if f, err := os.Stat(metaDir); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir(metaDir, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create metadata directory: %s\n", err.Error())
			os.Exit(1)
			return
		}
	}
	versionJSON, err := json.Marshal(request.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal version json: %s\n", err.Error())
		os.Exit(1)
		return
	}
	if err := ioutil.WriteFile(filepath.Join(metaDir, "version.json"), []byte(versionJSON), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write version file: %s\n", err.Error())
		os.Exit(1)
		return
	}
	for _, meta := range metadata {
		if err := ioutil.WriteFile(filepath.Join(metaDir, meta.Name), []byte(meta.Value), 0644); err != nil {
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
