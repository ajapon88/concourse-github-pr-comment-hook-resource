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

	infoEncoder := json.NewEncoder(os.Stderr)
	infoEncoder.SetIndent("", "    ")

	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode: %s\n", err.Error())
		os.Exit(1)
		return
	}

	if err := request.Source.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to validate source: %s\n", err.Error())
		os.Exit(1)
		return
	}

	triggerPhrase := regexp.MustCompile(request.Source.TriggerPhrase)
	allowUsers := make(map[string]struct{}, len(request.Source.AllowUsers))
	for _, user := range request.Source.AllowUsers {
		allowUsers[user] = struct{}{}
	}

	client, err := resource.CreateGithubClient(&request.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create github client: %s\n", err.Error())
		os.Exit(1)
		return
	}
	for _, team := range request.Source.AllowTeams {
		users, err := client.GetTeamMembers(team.Org, team.Slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get team %s/%s: %s\n", team.Org, team.Slug, err.Error())
			os.Exit(1)
			return
		}
		for _, user := range users {
			name := user.GetLogin()
			allowUsers[name] = struct{}{}
		}
	}

	response := Response{}

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
		// infoEncoder.Encode(pullRequest)
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
			commentUser := comment.GetUser().GetLogin()
			// fmt.Fprintf(os.Stderr, "CommentUser '%s'\n", commentUser)
			if _, ok := allowUsers[commentUser]; !ok {
				continue
			}
			if triggerPhrase.MatchString(comment.GetBody()) {
				// infoEncoder.Encode(comment)
				version := resource.Version{
					PR:          strconv.Itoa(pullRequest.GetNumber()),
					Commit:      pullRequest.GetHead().GetSHA(),
					CommentID:   strconv.FormatInt(comment.GetID(), 10),
					Comment:     comment.GetBody(),
					CommentedAt: comment.GetCreatedAt(),
				}
				fmt.Fprintf(os.Stderr, "Version:\n")
				infoEncoder.Encode(version)
				response = append(response, version)
			}
		}
	}
	sort.Sort(response)

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
