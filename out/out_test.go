package out_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/XenoPhex/go-tracker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/onsi/gomega/ghttp"

	"github.com/concourse/tracker-resource"
	"github.com/concourse/tracker-resource/out"
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

	Describe("integration with the real Tracker API", func() {
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

	})

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
				Params: out.Params{
					Repos: []string{
						"git",
						"middle/git2",
					},
				},
			}
			response = out.OutResponse{}

		})

		AfterEach(func() {
			server.Close()
		})

		Context("without a content file specified", func() {
			It("raises error", func() {
			})
		})

		Context("when a content file is specified with one story", func() {
			BeforeEach(func() {
				commentPath := "tracker-resource-content"
				request.Params.CommentPath = commentPath
				err := ioutil.WriteFile(filepath.Join(tmpdir, commentPath), []byte("some custom content"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				server.AppendHandlers(
					listStoriesHandler(trackerToken),
					storyActivityHandler(trackerToken, projectId, 565),

					storyActivityHandler(trackerToken, projectId, 123456),
					deliverStoryHandler(trackerToken, projectId, 123456),
					deliverStoryCommentHandler(trackerToken, projectId, 123456, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 123457),
					deliverStoryHandler(trackerToken, projectId, 123457),
					deliverStoryCommentHandler(trackerToken, projectId, 123457, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 223456),
					deliverStoryHandler(trackerToken, projectId, 223456),
					deliverStoryCommentHandler(trackerToken, projectId, 223456, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 323456),
					deliverStoryHandler(trackerToken, projectId, 323456),
					deliverStoryCommentHandler(trackerToken, projectId, 323456, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 423456),
					deliverStoryHandler(trackerToken, projectId, 423456),
					deliverStoryCommentHandler(trackerToken, projectId, 423456, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 523456),
					deliverStoryHandler(trackerToken, projectId, 523456),
					deliverStoryCommentHandler(trackerToken, projectId, 523456, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 789456),
					deliverStoryHandler(trackerToken, projectId, 789456),
					deliverStoryCommentHandler(trackerToken, projectId, 789456, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 223457),
					deliverStoryHandler(trackerToken, projectId, 223457),
					deliverStoryCommentHandler(trackerToken, projectId, 223457, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 323457),
					deliverStoryHandler(trackerToken, projectId, 323457),
					deliverStoryCommentHandler(trackerToken, projectId, 323457, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 423457),
					deliverStoryHandler(trackerToken, projectId, 423457),
					deliverStoryCommentHandler(trackerToken, projectId, 423457, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 444444),

					storyActivityHandler(trackerToken, projectId, 555555),
					deliverStoryHandler(trackerToken, projectId, 555555),
					deliverStoryCommentHandler(trackerToken, projectId, 555555, "some custom comment"),

					storyActivityHandler(trackerToken, projectId, 666666),
				)
			})

			It("should make a story with the file's contents", func() {
				session := runCommand(outCmd, request)
				Expect(session.Err).To(Say("Checking for finished story: .*#123456"))
				Expect(session.Err).To(Say("Checking for finished story: .*#123457"))

				os.Remove(request.Params.CommentPath)
			})
		})

		Context("when the comment file does not exist", func() {
			It("should return a fatal error", func() {
				request.Params.CommentPath = "some-non-existent-file"

				session := runCommandExpectingStatus(outCmd, request, 1)
				Expect(session.Err).To(Say("error reading comment file: open"))
			})
		})

		Context("when the activity endpoint returns an error", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					listStoriesHandler(trackerToken),
					ghttp.RespondWith(http.StatusInternalServerError, nil),
				)
			})

			It("returns a fatal error", func() {
				session := runCommandExpectingStatus(outCmd, request, 1)
				Expect(session.Err).To(Say("error fetching activity for story #565:"))
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
