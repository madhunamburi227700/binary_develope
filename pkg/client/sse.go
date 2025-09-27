package client

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
)

type SSEEvent struct {
	Data string
	Err  error
}

type SSEResponse struct {
	StatusCode int
	Headers    http.Header
	EventsChan chan SSEEvent
}

type SSEClient struct {
	RestClient *RESTClient
}

func NewSSEClient(restClient *RESTClient) *SSEClient {
	return &SSEClient{
		RestClient: restClient,
	}
}

func (c *SSEClient) SSERequest(ctx context.Context, endpoint string, method string, body interface{}, options *RequestOptions) (*SSEResponse, error) {

	req, err := c.RestClient.prepareRequest(ctx, method, endpoint, body, options)
	if err != nil {
		return nil, err
	}

	resp, err := c.RestClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	sse := SSEResponse{
		EventsChan: make(chan SSEEvent),
		Headers:    resp.Header,
		StatusCode: resp.StatusCode,
	}

	go func() {
		defer resp.Body.Close()
		defer close(sse.EventsChan)

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err.Error() == "EOF" {
						fmt.Println("Server closed the connection (EOF)")
					} else {
						fmt.Printf("Unexpected error: %v\n", err)
						sse.EventsChan <- SSEEvent{Err: err}
					}
					return
				}

				line = strings.TrimSpace(line)
				if len(line) == 0 {
					continue
				}
				sse.EventsChan <- SSEEvent{Data: line}
			}
		}
	}()

	return &sse, nil
}

func FlushSSE(ctx context.Context, w http.ResponseWriter, resp SSEResponse) error {

	headersSSE(w, resp.Headers)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("failed to cast ResponseWriter to Flusher")
	}

	for {
		select {
		case event, ok := <-resp.EventsChan:
			fmt.Println("Received event data", event.Data)
			if !ok {
				return nil
			}

			if event.Err != nil {
				return fmt.Errorf("Error reading SSE event: %v", event.Err)
			}

			_, err := fmt.Fprintf(w, "%s\n\n", event.Data)
			if err != nil {
				return fmt.Errorf("error forwarding SSE event: %v", err)
			}

			flusher.Flush()

		case <-ctx.Done():
			return nil
		}
	}
}

func headersSSE(w http.ResponseWriter, headers http.Header) {
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
}
