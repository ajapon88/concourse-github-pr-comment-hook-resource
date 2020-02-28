package resource

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Source struct {
	AccessToken   string   `json:"access_token"`
	Repository    string   `json:"repository"`
	TriggerPhrase string   `json:"trigger_phrase"`
	AllowUsers    []string `json:"allow_users"`
	AllowTeams    []struct {
		Org  string `json:"org"`
		Slug string `json:"slug"`
	} `json:"allow_teams"`
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
