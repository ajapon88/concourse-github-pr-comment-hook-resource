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

	client, err := resource.CreateGithubClient(&request.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create github client: %s\n", err.Error())
		os.Exit(1)
		return
	}
	allowUsers, err := getGithubUsers(client, request.Source.AllowUsers, request.Source.AllowTeams)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	ignoreUsers, err := getGithubUsers(client, request.Source.IgnoreUsers, request.Source.IgnoreTeams)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
		return
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
		lastCommentID, err = request.Version.GetCommentID()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
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
			if !request.Source.AllowAllUsers {
				if _, ok := allowUsers[commentUser]; !ok {
					continue
				}
			}
			if _, ok := ignoreUsers[commentUser]; ok {
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

func getGithubUsers(client *resource.GithubClient, users []string, teams []resource.Team) (map[string]struct{}, error) {
	userMap := make(map[string]struct{}, len(users))
	for _, user := range users {
		userMap[user] = struct{}{}
	}
	for _, team := range teams {
		users, err := client.GetTeamMembers(team.Organization, team.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to get team %s/%s: %s", team.Organization, team.Slug, err.Error())
		}
		for _, user := range users {
			name := user.GetLogin()
			userMap[name] = struct{}{}
		}
	}
	return userMap, nil
}
