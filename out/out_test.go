package out_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/XenoPhex/go-tracker"
	"github.com/cjcjameson/tracker-story-resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cjcjameson/tracker-story-resource/out"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Out", func() {
	var (
		outCmd *exec.Cmd
		tmpdir string
	)

	BeforeEach(func() {
		var err error

		tmpdir, err = ioutil.TempDir("", "out-tmp")
		Expect(err).NotTo(HaveOccurred())
		err = os.MkdirAll(tmpdir, 0755)
		Expect(err).NotTo(HaveOccurred())

		outCmd = exec.Command(outPath, tmpdir)
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	/*	Describe("integration with the real Tracker API", func() {
		var (
			request            out.OutRequest
			storyId            string
			projectId          string
			actualTrackerToken string
		)

		BeforeEach(func() {
			projectId = os.Getenv("TRACKER_PROJECT")
			if projectId == "" {
				Skip("TRACKER_PROJECT must be provided.")
			}

			actualTrackerToken = os.Getenv("TRACKER_TOKEN")
			if actualTrackerToken == "" {
				Skip("TRACKER_TOKEN must be provided.")
			}

			storyId = createActualStory(projectId, actualTrackerToken)
			setupTestEnvironmentWithActualStoryID(tmpdir, storyId)

			request = out.OutRequest{
				Source: resource.Source{
					Token:      actualTrackerToken,
					TrackerURL: "https://www.pivotaltracker.com",
					ProjectID:  projectId,
				},
				Params: out.Params{
					Repos: []string{
						"middle/git3",
					},
				},
			}
		})

		AfterEach(func() {
			deleteActualStory(projectId, actualTrackerToken, storyId)
		})

	})*/

	Context("when executed against a mock URL", func() {
		var request out.OutRequest
		var response out.OutResponse

		var server *ghttp.Server

		trackerToken := "abc"
		projectId := "1234"

		BeforeEach(func() {
			setupTestEnvironment(tmpdir)

			server = ghttp.NewServer()

			request = out.OutRequest{
				Source: resource.Source{
					Token:      trackerToken,
					TrackerURL: server.URL(),
					ProjectID:  projectId,
				},
				Params: out.Params{},
			}
			response = out.OutResponse{}
		})

		AfterEach(func() {
			server.Close()
		})

		FContext("without a content file specified", func() {
			It("raises error", func() {
				session := runCommandExpectingStatus(outCmd, request, 1)
				Expect(session.Err).To(Say("no content file specified"))
			})
		})

		FContext("when a content file is specified that doesn't exist", func() {
			It("raises error", func() {
				contentPath := "blah"
				request.Params.ContentPath = contentPath
				session := runCommandExpectingStatus(outCmd, request, 1)
				Expect(session.Err).To(Say("reading content file"))
			})
		})

		FContext("when a content file is specified with one story", func() {
			BeforeEach(func() {
				contentPath := "tracker-resource-content"
				request.Params.ContentPath = contentPath
				err := ioutil.WriteFile(filepath.Join(tmpdir, contentPath), []byte("fake-story-name"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				server.AppendHandlers(
					createStoryHandler(trackerToken, projectId),
				)
			})

			It("should make a story with the file's contents", func() {
				session := runCommand(outCmd, request)
				Expect(session.Err).To(Say("Story created with ID: 2300 Name: fake-story-name"))
				os.Remove(request.Params.ContentPath)
			})
		})
	})
})

func runCommand(outCmd *exec.Cmd, request out.OutRequest) *Session {
	return runCommandExpectingStatus(outCmd, request, 0)
}

func runCommandExpectingStatus(outCmd *exec.Cmd, request out.OutRequest, status int) *Session {
	timeout := 10 * time.Second
	stdin, err := outCmd.StdinPipe()
	Expect(err).NotTo(HaveOccurred())

	session, err := Start(outCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	err = json.NewEncoder(stdin).Encode(request)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, timeout).Should(Exit(status))

	return session
}

func createStoryHandler(token string, projectId string) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(
			"POST",
			fmt.Sprintf("/services/v5/projects/%s/stories", projectId),
		), ghttp.VerifyHeaderKV("X-TrackerToken", token),
		ghttp.RespondWith(http.StatusOK, Fixture("new_story.json")),
	)
}

func listStoriesHandler(trackerToken string) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/services/v5/projects/1234/stories"),
		ghttp.VerifyHeaderKV("X-TrackerToken", trackerToken),
		ghttp.RespondWith(http.StatusOK, Fixture("stories.json")),
	)
}

func setupTestEnvironment(path string) {
	setupTestEnvironmentWithActualStoryID(path, "")
}

func setupTestEnvironmentWithActualStoryID(path string, storyId string) {
	cmd := exec.Command(filepath.Join("scripts/setup.sh"), path, storyId)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
}

func createActualStory(projectID string, trackerToken string) string {
	projectIDInt, err := strconv.Atoi(projectID)
	Expect(err).NotTo(HaveOccurred())

	client := tracker.NewClient(trackerToken).InProject(projectIDInt)
	story := tracker.Story{
		Name:  "concourse test story",
		Type:  tracker.StoryTypeBug,
		State: tracker.StoryStateFinished,
	}
	story, err = client.CreateStory(story)
	Expect(err).NotTo(HaveOccurred())
	return strconv.Itoa(story.ID)
}

func deleteActualStory(projectID string, trackerToken string, storyId string) {
	projectIDInt, err := strconv.Atoi(projectID)
	Expect(err).NotTo(HaveOccurred())

	storyIDInt, err := strconv.Atoi(storyId)
	Expect(err).NotTo(HaveOccurred())

	client := tracker.NewClient(trackerToken).InProject(projectIDInt)
	err = client.DeleteStory(storyIDInt)
	Expect(err).NotTo(HaveOccurred())
}
