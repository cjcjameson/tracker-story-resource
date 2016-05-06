package in

import "github.com/cjcjameson/tracker-story-resource"

type InRequest struct {
	Source  resource.Source  `json:"source"`
	Version resource.Version `json:"version"`
}

type InResponse struct {
	Version  resource.Version        `json:"version"`
	Metadata []resource.MetadataPair `json:"metadata"`
}
