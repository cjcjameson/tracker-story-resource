package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type ProjectClient struct {
	id   int
	conn connection
}

func (p ProjectClient) Stories(query StoriesQuery) ([]Story, Pagination, error) {
	params := query.Query().Encode()

	request, err := p.createRequest("GET", "/stories?"+params)
	if err != nil {
		return nil, Pagination{}, err
	}

	var stories []Story
	pagination, err := p.conn.Do(request, &stories)
	if err != nil {
		return nil, Pagination{}, err
	}

	return stories, pagination, err
}

func (p ProjectClient) StoryActivity(storyId int, query ActivityQuery) (activities []Activity, err error) {
	url := fmt.Sprintf("/stories/%d/activity", storyId)
	params := query.Query().Encode()
	request, err := p.createRequest("GET", url+"?"+params)
	if err != nil {
		return activities, err
	}

	_, err = p.conn.Do(request, &activities)
	return activities, err
}

func (p ProjectClient) DeliverStoryWithComment(storyId int, comment string) error {
	err := p.DeliverStory(storyId)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/stories/%d/comments", storyId)
	request, err := p.createRequest("POST", url)
	if err != nil {
		return err
	}

	buffer := &bytes.Buffer{}
	json.NewEncoder(buffer).Encode(Comment{
		Text: comment,
	})

	p.addJSONBodyReader(request, buffer)

	_, err = p.conn.Do(request, nil)
	return err
}

func (p ProjectClient) DeliverStory(storyId int) error {
	url := fmt.Sprintf("/stories/%d", storyId)
	request, err := p.createRequest("PUT", url)
	if err != nil {
		return err
	}

	p.addJSONBody(request, `{"current_state":"delivered"}`)

	_, err = p.conn.Do(request, nil)
	return err
}

func (p ProjectClient) CreateStory(story Story) (Story, error) {
	request, err := p.createRequest("POST", "/stories")
	if err != nil {
		return Story{}, err
	}

	buffer := &bytes.Buffer{}
	json.NewEncoder(buffer).Encode(story)

	p.addJSONBodyReader(request, buffer)

	var createdStory Story
	_, err = p.conn.Do(request, &createdStory)
	return createdStory, err
}

func (p ProjectClient) DeleteStory(storyId int) error {
	url := fmt.Sprintf("/stories/%d", storyId)
	request, err := p.createRequest("DELETE", url)
	if err != nil {
		return err
	}

	_, err = p.conn.Do(request, nil)
	return err
}

func (p ProjectClient) createRequest(method string, path string) (*http.Request, error) {
	projectPath := fmt.Sprintf("/projects/%d%s", p.id, path)
	return p.conn.CreateRequest(method, projectPath)
}

func (p ProjectClient) addJSONBodyReader(request *http.Request, body io.Reader) {
	request.Header.Add("Content-Type", "application/json")
	request.Body = ioutil.NopCloser(body)
}

func (p ProjectClient) addJSONBody(request *http.Request, body string) {
	p.addJSONBodyReader(request, strings.NewReader(body))
}
