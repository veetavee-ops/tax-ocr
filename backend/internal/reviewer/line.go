package reviewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type LineClient struct {
	channelToken string
	httpClient   *http.Client
}

func NewLineClient(channelToken string) *LineClient {
	return &LineClient{
		channelToken: channelToken,
		httpClient:   &http.Client{},
	}
}

type pushPayload struct {
	To       string        `json:"to"`
	Messages []lineMessage `json:"messages"`
}

type lineMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Push sends a text message to a LINE user. No-op if token or userID is empty.
func (c *LineClient) Push(lineUserID, text string) error {
	if c.channelToken == "" || lineUserID == "" {
		return nil
	}
	payload := pushPayload{
		To:       lineUserID,
		Messages: []lineMessage{{Type: "text", Text: text}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.channelToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("line push: status %d", resp.StatusCode)
	}
	return nil
}

// Reply sends a reply using a one-time replyToken (must be called within 30s).
func (c *LineClient) Reply(replyToken, text string) error {
	if c.channelToken == "" || replyToken == "" {
		return nil
	}
	type replyPayload struct {
		ReplyToken string        `json:"replyToken"`
		Messages   []lineMessage `json:"messages"`
	}
	body, _ := json.Marshal(replyPayload{
		ReplyToken: replyToken,
		Messages:   []lineMessage{{Type: "text", Text: text}},
	})
	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/reply", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.channelToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("line reply: status %d", resp.StatusCode)
	}
	return nil
}

// GetContent downloads binary content for a LINE messageID (image, file).
func (c *LineClient) GetContent(messageID string) ([]byte, string, error) {
	if c.channelToken == "" {
		return nil, "", fmt.Errorf("LINE channel token not set")
	}
	req, err := http.NewRequest("GET",
		"https://api-data.line.me/v2/bot/message/"+messageID+"/content", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.channelToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("line get content: status %d", resp.StatusCode)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), resp.Header.Get("Content-Type"), nil
}
