package opencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client is an HTTP client for the OpenCode server API
type Client struct {
	serverURL  string
	username   string
	password   string
	modelID    string
	httpClient *http.Client
}

// Config holds configuration for the OpenCode client
type Config struct {
	ServerURL string
	Username  string
	Password  string
	ModelID   string
	Timeout   time.Duration
}

// NewClient creates a new OpenCode API client
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	return &Client{
		serverURL: strings.TrimRight(cfg.ServerURL, "/"),
		username:  cfg.Username,
		password:  cfg.Password,
		modelID:   cfg.ModelID,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateSessionRequest is the request body for creating a new session
type CreateSessionRequest struct {
	ModelID string `json:"model_id,omitempty"`
}

// CreateSessionResponse is the response from creating a new session
type CreateSessionResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

// SessionInfo contains detailed session information including token usage
type SessionInfo struct {
	ID               string  `json:"id"`
	Slug             string  `json:"slug"`
	ModelID          string  `json:"modelID"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	Cost             float64 `json:"cost"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
	SummaryMessageID string  `json:"summaryMessageID,omitempty"` // Set if session was compacted
}

// MessagePart represents a part of a message
type MessagePart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// SendMessageRequest is the request body for sending a message
type SendMessageRequest struct {
	Parts   []MessagePart `json:"parts"`
	ModelID string        `json:"model_id,omitempty"`
}

// ResponsePart represents a part of the response message
type ResponsePart struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
}

// MessageInfo contains metadata about the response message
type MessageInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	ModelID   string `json:"modelID"`
}

// SendMessageResponse is the response from sending a message
type SendMessageResponse struct {
	Info  MessageInfo    `json:"info"`
	Parts []ResponsePart `json:"parts"`
}

// Content extracts the text content from the response parts
func (r *SendMessageResponse) Content() string {
	for _, part := range r.Parts {
		if part.Type == "text" && part.Text != "" {
			return part.Text
		}
	}
	return ""
}

// SessionID returns the session ID from the response
func (r *SendMessageResponse) SessionID() string {
	return r.Info.SessionID
}

// GetSession retrieves session information including token usage
func (c *Client) GetSession(sessionID string) (*SessionInfo, error) {
	url := fmt.Sprintf("%s/session/%s", c.serverURL, sessionID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get session: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result SessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateSession creates a new OpenCode session
func (c *Client) CreateSession() (string, error) {
	url := c.serverURL + "/session"

	reqBody := CreateSessionRequest{
		ModelID: c.modelID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create session: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// SendMessage sends a message to an existing session
func (c *Client) SendMessage(sessionID, content string) (*SendMessageResponse, error) {
	url := fmt.Sprintf("%s/session/%s/message", c.serverURL, sessionID)

	reqBody := SendMessageRequest{
		Parts: []MessagePart{
			{Type: "text", Text: content},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to send message: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// setHeaders sets common headers including basic auth
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// ServerURL returns the configured server URL
func (c *Client) ServerURL() string {
	return c.serverURL
}

// ModelID returns the configured model ID
func (c *Client) ModelID() string {
	return c.modelID
}

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

// MessageUpdatedProps contains properties for message.updated events
type MessageUpdatedProps struct {
	Info struct {
		ID        string `json:"id"`
		SessionID string `json:"sessionID"`
		Role      string `json:"role"`
	} `json:"info"`
}

// PartUpdatedProps contains properties for message.part.updated events
type PartUpdatedProps struct {
	Part struct {
		ID        string `json:"id"`
		SessionID string `json:"sessionID"`
		MessageID string `json:"messageID"`
		Type      string `json:"type"`
		Text      string `json:"text"`
		// Tool-specific fields
		Tool   string `json:"tool,omitempty"`
		CallID string `json:"callID,omitempty"`
		State  *struct {
			Status string                 `json:"status,omitempty"`
			Input  map[string]interface{} `json:"input,omitempty"`
		} `json:"state,omitempty"`
	} `json:"part"`
}

// SessionStatusProps contains properties for session.status events
type SessionStatusProps struct {
	SessionID string `json:"sessionID"`
	Status    struct {
		Type    string `json:"type"`
		Attempt int    `json:"attempt,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"status"`
}

// SessionCompactedProps contains properties for session.compacted events
type SessionCompactedProps struct {
	SessionID        string `json:"sessionID"`
	SummaryMessageID string `json:"summaryMessageID"`
	TokensSaved      int    `json:"tokensSaved,omitempty"`
}

// SessionUpdatedProps contains properties for session.updated events (includes token counts)
type SessionUpdatedProps struct {
	Session SessionInfo `json:"session"`
}

// StreamEventCallback is called for each SSE event received
type StreamEventCallback func(event SSEEvent)

// SendMessageAsync sends a message asynchronously (non-blocking)
func (c *Client) SendMessageAsync(sessionID, content string) error {
	url := fmt.Sprintf("%s/session/%s/prompt_async", c.serverURL, sessionID)

	reqBody := SendMessageRequest{
		Parts: []MessagePart{
			{Type: "text", Text: content},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send async message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send async message: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// StreamResult contains the result of a streaming message
type StreamResult struct {
	Content          string
	SessionID        string
	MessageID        string
	Error            error
	RetryCount       int  // Number of API retries during streaming
	WasCompacted     bool // True if session was compacted during this message
	PromptTokens     int  // Token usage after message
	CompletionTokens int  // Token usage after message
}

// SendMessageStreaming sends a message and streams the response via SSE
func (c *Client) SendMessageStreaming(ctx context.Context, sessionID, content string, eventCb StreamEventCallback) (*StreamResult, error) {
	// Create a dedicated HTTP client for SSE with no timeout
	sseClient := &http.Client{}

	// Start SSE connection
	sseReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL+"/event", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE request: %w", err)
	}

	sseReq.Header.Set("Accept", "text/event-stream")
	sseReq.Header.Set("Cache-Control", "no-cache")
	if c.username != "" && c.password != "" {
		sseReq.SetBasicAuth(c.username, c.password)
	}

	sseResp, err := sseClient.Do(sseReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSE: %w", err)
	}

	result := &StreamResult{SessionID: sessionID}
	var mu sync.Mutex

	// Track assistant message parts by messageID
	assistantMsgID := ""
	parts := make(map[string]string) // partID -> text
	done := make(chan struct{})
	errChan := make(chan error, 1)
	retryCount := 0
	maxRetries := 20 // Maximum number of API retries before giving up

	// Start SSE reader goroutine
	go func() {
		defer sseResp.Body.Close()

		reader := bufio.NewReader(sseResp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					select {
					case errChan <- fmt.Errorf("SSE read error: %w", err):
					default:
					}
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			var event SSEEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if eventCb != nil {
				eventCb(event)
			}

			mu.Lock()
			switch event.Type {
			case "message.updated":
				var props MessageUpdatedProps
				if err := json.Unmarshal(event.Properties, &props); err == nil {
					if props.Info.Role == "assistant" && props.Info.SessionID == sessionID {
						assistantMsgID = props.Info.ID
						result.MessageID = assistantMsgID
					}
				}

			case "message.part.updated":
				var props PartUpdatedProps
				if err := json.Unmarshal(event.Properties, &props); err == nil {
					if props.Part.SessionID == sessionID && props.Part.MessageID == assistantMsgID {
						if props.Part.Type == "text" {
							parts[props.Part.ID] = props.Part.Text
						}
					}
				}

			case "session.status":
				var props SessionStatusProps
				if err := json.Unmarshal(event.Properties, &props); err == nil {
					if props.SessionID == sessionID {
						switch props.Status.Type {
						case "idle":
							// Session is done processing
							result.RetryCount = retryCount
							mu.Unlock()
							close(done)
							return
						case "retry":
							retryCount = props.Status.Attempt
							if retryCount > maxRetries {
								mu.Unlock()
								select {
								case errChan <- fmt.Errorf("API rate limited after %d retries: %s", retryCount, props.Status.Message):
								default:
								}
								return
							}
						case "error":
							mu.Unlock()
							select {
							case errChan <- fmt.Errorf("session error: %s", props.Status.Message):
							default:
							}
							return
						}
					}
				}

			case "session.compacted":
				var props SessionCompactedProps
				if err := json.Unmarshal(event.Properties, &props); err == nil {
					if props.SessionID == sessionID {
						result.WasCompacted = true
					}
				}

			case "session.updated":
				var props SessionUpdatedProps
				if err := json.Unmarshal(event.Properties, &props); err == nil {
					if props.Session.ID == sessionID {
						result.PromptTokens = props.Session.PromptTokens
						result.CompletionTokens = props.Session.CompletionTokens
					}
				}
			}
			mu.Unlock()
		}
	}()

	// Wait a moment for SSE to be ready, then send the message
	time.Sleep(100 * time.Millisecond)

	if err := c.SendMessageAsync(sessionID, content); err != nil {
		sseResp.Body.Close() // Close SSE to unblock reader
		return nil, err
	}

	// Wait for completion, error, or context cancellation
	select {
	case <-done:
		// Success - session went idle
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		sseResp.Body.Close() // Close SSE to unblock reader
		return nil, ctx.Err()
	}

	// Combine all text parts
	mu.Lock()
	var texts []string
	for _, text := range parts {
		if text != "" {
			texts = append(texts, text)
		}
	}
	result.Content = strings.Join(texts, "\n")
	mu.Unlock()

	return result, nil
}
