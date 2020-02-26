package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"

	"github.com/ajapon88/concourse-github-pr-comment-hook-resource"
)

type Request struct {
	Source  resource.Source  `json:"source"`
	Version resource.Version `json:"version"`
}

type Response []resource.Version

func (r Response) Len() int {
	return len(r)
}

func (r Response) Less(i, j int) bool {
	return r[i].CommentID < r[j].CommentID
}

func (r Response) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func main() {
	var request Request
	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode: %s\n", err.Error())
		os.Exit(1)
		return
	}

	fmt.Fprintf(os.Stderr, "source: %v\n", request.Source)

	if request.Source.TriggerPhrase == "" {
		fmt.Fprintf(os.Stderr, "trigger phrase must be set\n")
		os.Exit(1)
		return
	}
	triggerPhrase := regexp.MustCompile(request.Source.TriggerPhrase)

	response := Response{}

	client, err := resource.CreateGithubClient(&request.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create github client: %s\n", err.Error())
		os.Exit(1)
		return
	}

	pullRequests, err := client.GetListPullRequests()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get PullRequests: %s\n", err.Error())
		os.Exit(1)
		return
	}
	var lastCommentID int64
	if request.Version.CommentID != "" {
		lastCommentID, err = strconv.ParseInt(request.Version.CommentID, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to atoi CommentID: %s\n", err.Error())
			os.Exit(1)
			return
		}
	} else {
		lastCommentID = 0
	}
	for _, pullRequest := range pullRequests {
		// fmt.Fprintf(os.Stderr, "--\n%+v\n", pullRequest)
		comments, err := client.GetListIssueComments(pullRequest.GetNumber())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get comments: %s\n", err.Error())
			os.Exit(1)
			return
		}
		for _, comment := range comments {
			if comment.GetID() <= lastCommentID {
				continue
			}
			if triggerPhrase.MatchString(comment.GetBody()) {
				//fmt.Fprintf(os.Stderr, "trigger:\n        PR: %d\n        ID: %d\n      Body: %s\n CreatedAt: %s\n", pullRequest.GetNumber(), comment.GetID(), comment.GetBody(), comment.GetCreatedAt())
				version := resource.Version{
					PR:        strconv.Itoa(pullRequest.GetNumber()),
					CommentID: strconv.FormatInt(comment.GetID(), 10),
					CreatedAt: comment.GetCreatedAt(),
				}
				response = append(response, version)
			}
		}
	}
	sort.Sort(response)
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "    ")
	enc.Encode(response)

	// チェックするコメントがなければversionをそのまま返す
	if len(response) == 0 && request.Version.CommentID != "" {
		response = Response{request.Version}
	}
	// versionが空（一番最初or手動起動時）は最新のバージョンにする
	if len(response) != 0 && request.Version.CommentID == "" {
		response = Response{response[len(response)-1]}
	}
	if len(response) != 0 {
		response = Response{response[0]}
	}

	json.NewEncoder(os.Stdout).Encode(response)
}
