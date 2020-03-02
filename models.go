package resource

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Source struct {
	AccessToken   string   `json:"access_token"`
	Repository    string   `json:"repository"`
	TriggerPhrase string   `json:"trigger_phrase"`
	AllowUsers    []string `json:"allow_users"`
	AllowTeams    []Team   `json:"allow_teams"`
	IgnoreUsers   []string `json:"ignore_users"`
	IgnoreTeams   []Team   `json:"ignore_teams"`
}

type Team struct {
	Organization string `json:"organization"`
	Slug         string `json:"slug"`
}

type Version struct {
	PR          string    `json:"pr"`
	Commit      string    `json:"commit"`
	CommentID   string    `json:"comment_id"`
	Comment     string    `json:"comment"`
	CommentedAt time.Time `json:"commented_at"`
}

type MetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Metadata []*MetadataField

func (source *Source) Validate() error {
	if source.AccessToken == "" {
		return fmt.Errorf("access_token must be set")
	}
	if source.Repository == "" {
		return fmt.Errorf("repository must be set")
	}
	if source.TriggerPhrase == "" {
		return fmt.Errorf("trigger_phrase must be set")
	}
	if len(source.AllowUsers) == 0 && len(source.AllowTeams) == 0 {
		return fmt.Errorf("allow_users or allow_teams must be set")
	}
	return nil
}

func (source *Source) GetOwnerRepo() (string, string, error) {
	slice := strings.Split(source.Repository, "/")
	if len(slice) != 2 {
		return "", "", errors.New("invalid repository format")
	}
	return slice[0], slice[1], nil
}

func (version *Version) GetPR() (int, error) {
	number, err := strconv.Atoi(version.PR)
	if err != nil {
		return 0, fmt.Errorf("failed to atoi pr number '%s': %s", version.PR, err.Error())
	}
	return number, nil
}

func (version *Version) GetCommentID() (int64, error) {
	id, err := strconv.ParseInt(version.CommentID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CommentID '%s': %s", version.CommentID, err.Error())
	}
	return id, nil
}
