package resource

import (
	"context"

	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	Client *github.Client
	Repo   string
	Owner  string
}

func CreateGithubClient(source *Source) (*GithubClient, error) {
	owner, repo, err := source.GetOwnerRepo()
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: source.AccessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	return &GithubClient{
		Client: client,
		Repo:   repo,
		Owner:  owner,
	}, nil
}

func (client *GithubClient) GetPullRequest(number int) (*github.PullRequest, error) {
	pullRequest, _, err := client.Client.PullRequests.Get(context.TODO(), client.Owner, client.Repo, number)
	if err != nil {
		return nil, err
	}

	return pullRequest, nil
}

func (client *GithubClient) GetListPullRequests() ([]*github.PullRequest, error) {
	var pullRequests []*github.PullRequest
	opts := &github.PullRequestListOptions{}

	for {
		pulls, resp, err := client.Client.PullRequests.List(context.TODO(), client.Owner, client.Repo, opts)
		if err != nil {
			return nil, err
		}
		pullRequests = append(pullRequests, pulls...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return pullRequests, nil
}

func (client *GithubClient) GetListIssueComments(number int) ([]*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{}
	var comments []*github.IssueComment

	for {
		cmnts, resp, err := client.Client.Issues.ListComments(context.TODO(), client.Owner, client.Repo, number, opts)
		if err != nil {
			return nil, err
		}
		comments = append(comments, cmnts...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return comments, nil
}

func (client *GithubClient) GetListPullRequestCommits(number int) ([]*github.RepositoryCommit, error) {
	var commits []*github.RepositoryCommit
	opts := &github.ListOptions{}

	for {
		cmts, resp, err := client.Client.PullRequests.ListCommits(context.TODO(), client.Owner, client.Repo, number, opts)
		if err != nil {
			return nil, err
		}
		commits = append(commits, cmts...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return commits, nil
}
