package resource

import (
	"errors"
	"strings"
	"time"
)

type Source struct {
	AccessToken   string `json:"access_token"`
	Repository    string `json:"repository"`
	TriggerPhrase string `json:"trigger_phrase"`
}

type Version struct {
	PR        string    `json:"pr"`
	CommentID string    `json:"comment_id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_date"`
}

type MetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (source *Source) GetOwnerRepo() (string, string, error) {
	slice := strings.Split(source.Repository, "/")
	if len(slice) != 2 {
		return "", "", errors.New("invalid repository format")
	}
	return slice[0], slice[1], nil
}
