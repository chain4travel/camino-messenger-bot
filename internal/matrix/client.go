package matrix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
)

const (
	DefaultInterval            = 10000 * time.Millisecond
	ThirdPartyIdentifierMedium = "camino"
)

var (
	_ Client = (*client)(nil)
)

type Session struct {
	AccessToken string `json:"access_token"`
	UserId      string `json:"user_id"`
	FilterId    string `json:"filter_id"`
	NextBatch   string `json:"next_batch"`
}

type LoginRequestIdentifier struct {
	Type    string `json:"type"`
	Medium  string `json:"medium"`
	Address string `json:"address"`
}
type LoginRequest struct {
	Type       string                 `json:"type"`
	Identifier LoginRequestIdentifier `json:"identifier"`
	Password   string                 `json:"password"`
}
type client struct {
	matrixHost      string
	pollingInterval time.Duration
	logger          *zap.SugaredLogger
	session         Session
}

func (c *client) Upload(payload, eventID string) error {
	//TODO implement me
	panic("implement me")
}

func (c *client) Join(roomID string) error {
	//TODO implement me
	panic("implement me")
}

func (c *client) Send(roomID string, message TimelineEventContent) error {
	//TODO implement me
	//panic("implement me")

	c.logger.Debug("Matrix send called")
	return nil

}

type Client interface {
	Login(matrixLoginRequest LoginRequest) error
	Sync(roomInviteChannel chan<- InviteRooms, roomTimelineChannel chan<- JoinedRooms)
	Upload(payload, eventID string) error
	Join(roomID string) error
	Send(roomID string, message TimelineEventContent) error

	getOrInitSyncFilter() (string, error)
}

func NewClient(matrixHost string, logger *zap.SugaredLogger, pollingInterval time.Duration) Client {
	return &client{matrixHost: matrixHost, logger: logger, pollingInterval: pollingInterval}
}
func (c *client) Login(matrixLoginRequest LoginRequest) error {
	url := fmt.Sprintf("https://%s/_matrix/client/v3/login", c.matrixHost)

	requestBodyJSON, err := json.Marshal(matrixLoginRequest)
	if err != nil {
		return fmt.Errorf("error while serializing request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBodyJSON))
	if err != nil {
		return fmt.Errorf("error while making the HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return fmt.Errorf("Error while decoding the response: %v\n", err)
		}

		c.session.AccessToken = response["access_token"].(string)
		c.session.UserId = response["user_id"].(string)

	} else {

		// error due to rate limiting
		if resp.StatusCode == 429 {
			log.Printf("rate limiting")
			var errorResponse map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&errorResponse)
			if err != nil {
				return fmt.Errorf("Error while decoding the error response: %v\n", err)
			}

			sleepTime := int(errorResponse["retry_after_ms"].(float64))
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)

			c.Login(matrixLoginRequest) // Recursive call
		} else {
			return fmt.Errorf("Error while logging in. Received status code: %d\n", resp.StatusCode)
		}
	}
	return nil
}

func (c *client) Sync(roomInviteChannel chan<- InviteRooms, roomTimelineChannel chan<- JoinedRooms) {
	filter, err := c.getOrInitSyncFilter()
	if err != nil {
		c.logger.Errorf("Error: %v\n", err)
		return
	}

	url := fmt.Sprintf("https://%s/_matrix/client/v3/sync?access_token=%s&filter=%s&timeout=%d", c.matrixHost, c.session.AccessToken, filter, c.pollingInterval/time.Millisecond)

	url = c.addNextBatchParam(url)
	c.logger.Infof("Syncing... %s\n", url)
	// Perform a GET request to the REST endpoint
	resp, err := http.Get(url)
	if err != nil {
		c.logger.Errorf("Error: %v\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		//body, err := io.ReadAll(resp.Body)
		//if err != nil {
		//	c.logger.Errorf("Error reading response body: %v\n", err)
		//}

		// Process the response body here
		var bodyResponse SyncResponse
		err = json.NewDecoder(resp.Body).Decode(&bodyResponse)
		if err != nil {
			c.logger.Errorf("Error reading response body: %v\n", err)
		}
		c.session.NextBatch = bodyResponse.NextBatch

		if bodyResponse.Rooms.Invite != nil {
			roomInviteChannel <- bodyResponse.Rooms.Invite
		}
		if bodyResponse.Rooms.Join != nil {
			roomTimelineChannel <- bodyResponse.Rooms.Join
		}
		c.logger.Infof("Next batch: %s\n", c.session.NextBatch)
	} else {
		var errorResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			c.logger.Errorf("Error while decoding the error response: %v\n", err)
		}

		c.logger.Errorf("Failed to sync - status code: %d\n", resp.StatusCode)
	}
}

func (c *client) getOrInitSyncFilter() (string, error) {
	if c.session.FilterId != "" {
		return c.session.FilterId, nil
	}
	url := fmt.Sprintf("https://%s/_matrix/client/v3/user/%s/filter?access_token=%s", c.matrixHost, c.session.UserId, c.session.AccessToken)

	requestBodyJSON := []byte(`{
                    "account_data": {
                        "not_types": ["*"]
                    },
                    "presence": {
                        "not_types": ["*"]
                    },
                    "room" : {
                        "state": {
                            "types": ["m.room.*"]
                        },
                        "ephemeral": {
                            "not_types": ["*"]
                        }
                    }
                }`)
	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBodyJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return "", err
		}

		c.session.FilterId = response["filter_id"].(string)
		c.logger.Infof("Created filter id: %s", c.session.FilterId)
		return c.session.FilterId, nil
	} else {

		// error due to rate limiting
		if resp.StatusCode == 429 {
			log.Printf("rate limiting")
			var errorResponse map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&errorResponse)
			if err != nil {
				return "", fmt.Errorf("Error while decoding the error response: %v\n", err)
			}

			sleepTime := int(errorResponse["retry_after_ms"].(float64))
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)

			return c.getOrInitSyncFilter() // Recursive call
		} else {
			return "", fmt.Errorf("Error while logging in. Received status code: %d\n", resp.StatusCode)
		}
	}
	log.Printf("Received non-200 status code: %d\n", resp.StatusCode)
	return "", fmt.Errorf("Received non-200 status code: %d", resp.StatusCode)
}

func (c *client) addNextBatchParam(url string) string {
	if c.session.NextBatch != "" {
		return fmt.Sprintf("%s&since=%s", url, c.session.NextBatch)
	}
	return url
}
