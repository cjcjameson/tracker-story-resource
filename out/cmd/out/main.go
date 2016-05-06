package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/XenoPhex/go-tracker"
	"github.com/cjcjameson/tracker-story-resource"
	"github.com/cjcjameson/tracker-story-resource/out"
	"github.com/mitchellh/colorstring"
)

func buildRequest() out.OutRequest {
	var request out.OutRequest

	err := json.NewDecoder(os.Stdin).Decode(&request)
	if err != nil {
		fatal("reading request", err)
	}

	return request
}

func main() {
	if len(os.Args) < 2 {
		sayf("usage: %s <sources directory>\n", os.Args[0])
		os.Exit(1)
	}

	sources := os.Args[1]
	request := buildRequest()

	trackerURL := request.Source.TrackerURL
	if trackerURL == "" {
		trackerURL = "https://www.pivotaltracker.com"
	}

	token := request.Source.Token
	projectID, err := strconv.Atoi(request.Source.ProjectID)
	if err != nil {
		fatal("converting the project ID to an integer", err)
	}

	contentPath := request.Params.ContentPath

	var contents []byte
	if contentPath != "" {
		contents, err = ioutil.ReadFile(filepath.Join(sources, contentPath))
		if err != nil {
			fatal("reading content file", err)
		}
	} else {
		fatal("error", errors.New("no content file specified"))
	}

	tracker.DefaultURL = trackerURL
	client := tracker.NewClient(token).InProject(projectID)
	story := tracker.Story{
		Name:  string(contents),
		Type:  tracker.StoryTypeChore,
		State: tracker.StoryStateUnscheduled,
	}
	story, err = client.CreateStory(story)
	sayf("Story created with ID: %d Name: %s", story.ID, story.Name)

	outputResponse()
}

func outputResponse() {
	json.NewEncoder(os.Stdout).Encode(out.OutResponse{
		Version: resource.Version{
			Time: time.Now(),
		},
	})
}

func sayf(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
}

func fatal(doing string, err error) {
	sayf(colorstring.Color("[red]error %s: %s\n"), doing, err)
	os.Exit(1)
}
