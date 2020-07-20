package quay

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/joshvanl/version-checker/pkg/api"
	"github.com/joshvanl/version-checker/pkg/version/util"
)

const (
	repoURL     = "https://quay.io/api/v1/repository/%s"
	imagePrefix = "quay.io/"
)

var _ api.ImageClient = &Client{}

type Client struct {
	*http.Client
}

type Response struct {
	Tags map[string]Tag `json:"tags"`
}

type Tag struct {
	ManifestDigest string `json:"manifest_digest"`
	LastModified   string `json:"last_modified"`
}

func New() *Client {
	return &Client{
		Client: http.DefaultClient,
	}
}

func (c *Client) IsClient(imageURL string) bool {
	return strings.HasPrefix(imageURL, imagePrefix)
}

func (c *Client) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	if !c.IsClient(imageURL) {
		return nil, fmt.Errorf("image does not have %q prefix: %s", imagePrefix, imageURL)
	}

	url := fmt.Sprintf(repoURL, strings.TrimPrefix(imageURL, imagePrefix))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get quay image: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var tags []api.ImageTag
	for tagName, tag := range response.Tags {
		timestamp, err := time.Parse(time.RFC1123Z, tag.LastModified)
		if err != nil {
			return nil, err
		}

		shaID, err := util.ParseSHADigest(tag.ManifestDigest)
		if err != nil {
			return nil, err
		}

		tags = append(tags, api.ImageTag{
			Tag:       tagName,
			SHA:       shaID,
			Timestamp: timestamp,
		})
	}

	return tags, nil
}
