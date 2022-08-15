package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type client struct {
	AccessToken   string
	Conversations []Conversation
	httpClient    http.Client
}

type Conversation struct {
	ID           int    `json:"id"`
	Number       int    `json:"number"`
	ThreadsCount int    `json:"threads"`
	Type         string `json:"type"`
	FolderID     int    `json:"folderId"`
	Status       string `json:"status"`
	State        string `json:"state"`
	Subject      string `json:"subject"`
	Preview      string `json:"preview"`
	MailboxID    int    `json:"mailboxId"`
	CreatedBy    struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	ClosedByUser struct {
		ID    int    `json:"id"`
		Type  string `json:"type"`
		First string `json:"first"`
		Last  string `json:"last"`
		Email string `json:"email"`
	} `json:"closedByUser"`
	UserUpdatedAt time.Time `json:"userUpdatedAt"`
	Source        struct {
		Type string `json:"type"`
		Via  string `json:"via"`
	} `json:"source"`
	Cc              []string `json:"cc"`
	Bcc             []string `json:"bcc"`
	PrimaryCustomer struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"primaryCustomer"`
	Threads []Thread `json:"threads_data"`
}

type ConversationAPIResp struct {
	Data struct {
		Conversations []Conversation `json:"conversations"`
	} `json:"_embedded"`
	Links NextLink `json:"_links"`
}

type Thread struct {
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	State  string `json:"state"`
	Action struct {
		Type               string `json:"type"`
		Text               string `json:"text"`
		AssociatedEntities struct {
		} `json:"associatedEntities"`
	} `json:"action"`
	Body   string `json:"body"`
	Source struct {
		Type string `json:"type"`
		Via  string `json:"via"`
	} `json:"source"`
	Customer struct {
		ID       int    `json:"id"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"customer"`
	CreatedBy struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		First    string `json:"first"`
		Last     string `json:"last"`
		PhotoURL string `json:"photoUrl"`
		Email    string `json:"email"`
	} `json:"createdBy"`
	AssignedTo struct {
		ID    int    `json:"id"`
		Type  string `json:"type"`
		First string `json:"first"`
		Last  string `json:"last"`
		Email string `json:"email"`
	} `json:"assignedTo"`
	SavedReplyID int       `json:"savedReplyId"`
	To           []string  `json:"to"`
	Cc           []string  `json:"cc"`
	Bcc          []string  `json:"bcc"`
	CreatedAt    time.Time `json:"createdAt"`
	OpenedAt     time.Time `json:"openedAt"`
	Embedded     struct {
		Attachments []struct {
			ID       int    `json:"id"`
			Filename string `json:"filename"`
			MimeType string `json:"mimeType"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Size     int    `json:"size"`
			Links    struct {
				Self struct {
					Href string `json:"href"`
				} `json:"self"`
				Data struct {
					Href string `json:"href"`
				} `json:"data"`
				Web struct {
					Href string `json:"href"`
				} `json:"web"`
			} `json:"_links"`
		} `json:"attachments"`
	} `json:"_embedded"`
}

type ThreadsAPIResp struct {
	Data struct {
		Threads []Thread `json:"threads"`
	} `json:"_embedded"`
	Links NextLink `json:"_links"`
}

type NextLink struct {
	Next struct {
		Href string `json:"href"`
	} `json:"next"`
}

func main() {
	var (
		accessToken = flag.String("at", "", "Gets indirect modules as well")
	)

	flag.Parse()

	if accessToken == nil {
		log.Fatal("and accessToken (at) must be provided")
	}

	c := client{AccessToken: fmt.Sprintf("Bearer %s", *accessToken), httpClient: http.Client{Timeout: 10 * time.Second}}

	url := "https://api.helpscout.net/v2/conversations?status=all"
	var err error

	for {
		url, err = c.ListConversations(url)
		if err != nil {
			log.Fatal(err)
		}
		if url == "" {
			break
		}
	}
	bts, err := json.Marshal(c.Conversations)
	checkErr(err)

	checkErr(os.WriteFile("conversations.json", bts, os.ModePerm))

}

func (c *client) ListConversations(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", c.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Println("Doing: ", err, resp)
		return "", err
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		waitSecondsHeader := resp.Header.Get("X-RateLimit-Retry-After")
		seconds, _ := strconv.Atoi(waitSecondsHeader)
		if seconds == 0 {
			fmt.Println("waitSeconds was 0: ", waitSecondsHeader)
			seconds = 60
		}
		fmt.Printf("Sleeping for %d seconds due to rate limiting", seconds)
		time.Sleep(time.Second * time.Duration(seconds))
		return url, nil
	default:
		return "", errors.New(string(bts))
	}

	var respBody ConversationAPIResp
	err = json.Unmarshal(bts, &respBody)
	if err != nil {
		return "", err
	}

	fmt.Printf("Fetched %d conversations. Fetching threads now...\n", len(respBody.Data.Conversations))

	for i, conv := range respBody.Data.Conversations {
		fmt.Printf("Fetching threads for conversation %d...\n [%d/%d]",
			conv.ID, i+1, len(respBody.Data.Conversations))
		threads, err := c.ListThreads(conv.ID)
		checkErr(err)
		respBody.Data.Conversations[i].Threads = threads
	}

	c.Conversations = append(c.Conversations, respBody.Data.Conversations...)

	// Not sure if API returns Next if there is no Next page
	if respBody.Links.Next.Href == url {
		return "", nil
	}

	return respBody.Links.Next.Href, nil
}

func (c *client) ListThreads(conversationID int) ([]Thread, error) {
	url := fmt.Sprintf("https://api.helpscout.net/v2/conversations/%d/threads", conversationID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", c.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		waitSecondsHeader := resp.Header.Get("X-RateLimit-Retry-After")
		seconds, _ := strconv.Atoi(waitSecondsHeader)
		if seconds == 0 {
			fmt.Println("waitSeconds was 0: ", waitSecondsHeader)
			seconds = 60
		}
		fmt.Printf("Sleeping for %d seconds due to rate limiting", seconds)
		time.Sleep(time.Second * time.Duration(seconds))
		return c.ListThreads(conversationID)
	default:
		return nil, errors.New(string(bts))
	}

	var respBody ThreadsAPIResp
	err = json.Unmarshal(bts, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody.Data.Threads, nil
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
