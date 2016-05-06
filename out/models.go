package out

import "github.com/cjcjameson/tracker-story-resource"

type OutRequest struct {
	Source resource.Source `json:"source"`
	Params Params          `json:"params"`
}

type Params struct {
	ContentPath string `json:"content"`
}

type OutResponse struct {
	Version resource.Version `json:"version"`
}
