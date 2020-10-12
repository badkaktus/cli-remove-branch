package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/badkaktus/gorocket"
)

var glURL, glToken, rocketURL, rocketUser, rocketPassword, rocketChannel *string
var glProject *int
var glFullURL, rFullURL string
var client *http.Client
var wg sync.WaitGroup
var rocketClient *gorocket.Client

type SingleIssue struct {
	ID        int       `json:"id"`
	Iid       int       `json:"iid"`
	ProjectID int       `json:"project_id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  time.Time `json:"closed_at"`
}

type Branch struct {
	Name   string `json:"name"`
	Commit struct {
		ID             string      `json:"id"`
		ShortID        string      `json:"short_id"`
		CreatedAt      time.Time   `json:"created_at"`
		ParentIds      interface{} `json:"parent_ids"`
		Title          string      `json:"title"`
		Message        string      `json:"message"`
		AuthorName     string      `json:"author_name"`
		AuthorEmail    string      `json:"author_email"`
		AuthoredDate   time.Time   `json:"authored_date"`
		CommitterName  string      `json:"committer_name"`
		CommitterEmail string      `json:"committer_email"`
		CommittedDate  time.Time   `json:"committed_date"`
	} `json:"commit"`
	Merged             bool `json:"merged"`
	Protected          bool `json:"protected"`
	DevelopersCanPush  bool `json:"developers_can_push"`
	DevelopersCanMerge bool `json:"developers_can_merge"`
	CanPush            bool `json:"can_push"`
	Default            bool `json:"default"`
}

func main() {
	glURL = flag.String("glurl", "", "GitLab URL")
	glToken = flag.String("gltoken", "", "GitLab Private Token")
	glProject = flag.Int("glproject", 0, "GitLab Project ID")
	rocketURL = flag.String("rurl", "", "RocketChat URL")
	rocketUser = flag.String("ruser", "", "RocketChat User")
	rocketPassword = flag.String("rpass", "", "RocketChat Password")
	rocketChannel = flag.String("rch", "", "RocketChat channel to post")
	flag.Parse()

	glFullURL = setURL(*glURL)
	rFullURL = setURL(*rocketURL)

	client = &http.Client{}
	rocketClient = gorocket.NewClient(rFullURL)

	payload := gorocket.LoginPayload{
		User:     *rocketUser,
		Password: *rocketPassword,
	}

	loginResp, _ := rocketClient.Login(&payload)
	log.Printf("Rocket login status: %s", loginResp.Status)
	if loginResp.Message != "" {
		log.Printf("Rocket login response message: %s", loginResp.Message)
	}

	page := 1
	for {
		issuesURL := fmt.Sprintf(
			"%s/api/v4/projects/%d/issues?state=closed&page=%d&per_page=100&order_by=updated_at&sort=desc",
			glFullURL,
			*glProject,
			page,
		)
		code, body := httpHelper(
			"GET",
			issuesURL)
		allIssues := []SingleIssue{}

		log.Printf("Status code %d get issues", code)
		json.Unmarshal(body, &allIssues)
		log.Printf("Page %d, len of issues %d", page, len(allIssues))

		if len(allIssues) == 0 {
			log.Println("Thats all forks!!")
			break
		}

		for _, v := range allIssues {
			wg.Add(1)

			go func() {
				branchName := isBranchExist(v.Iid)
				if branchName != "" {
					log.Printf("Branch for issue %d exist. Kill them all!! %s", v.Iid, branchName)
					delBranch(branchName, v.Title, v.Iid)
				}
				wg.Done()
			}()

			wg.Wait()
		}

		page++
	}
}

func isBranchExist(iid int) string {
	url := fmt.Sprintf("%s/api/v4/projects/%d/repository/branches?search=^%d-", glFullURL, *glProject, iid)

	code, body := httpHelper(
		"GET",
		url)

	if code >= 300 {
		log.Printf("Error request. Status code: %d", code)
	}

	branches := []Branch{}

	json.Unmarshal(body, &branches)

	if len(branches) > 0 {
		return branches[0].Name
	}

	return ""
}

func delBranch(branchName, issueTitle string, issueId int) {
	url := fmt.Sprintf("%s/api/v4/projects/%d/repository/branches/%s", glFullURL, *glProject, branchName)
	log.Printf("Delete branch: %s", branchName)
	status, _ := httpHelper("DELETE", url)

	log.Printf("Return status code %d after delete branch\n", status)
	if status < 300 {

		opt := gorocket.Message{
			Text: fmt.Sprintf(
				"CLI tool: :computer: Issue with ID %d \"%s\" was closed and branch \"%s\" deleted",
				issueId,
				issueTitle,
				branchName,
			),
			Channel: *rocketChannel,
		}

		//log.Printf("Send message: \"%s\" to Rocket.Chat", opt.Text)

		hresp, err := rocketClient.PostMessage(&opt)

		//log.Printf("PostMessage response status: %v", hresp.Success)

		if err != nil || hresp.Success == false {
			log.Printf("Sending message to Rocket.Chat error")
		}
	}
}

func httpHelper(method, url string) (int, []byte) {
	req, err := http.NewRequest(method, url, nil)

	req.Header.Add("Private-Token", *glToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		panic(err)
		//log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		log.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)

	return res.StatusCode, body
}

func setURL(argURL string) string {
	if strings.HasPrefix(argURL, "http://") || strings.HasPrefix(argURL, "https://") {
		return argURL
	}

	return fmt.Sprintf("http://%s", argURL)
}
