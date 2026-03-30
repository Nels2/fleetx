package externalsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// FreeScout is a FreeScout client to be used to make requests to the FreeScout external service.
type FreeScout struct {
	client *http.Client
	opts   FreeScoutOptions
}

// FreeScoutOptions defines the options to configure a FreeScout client.
type FreeScoutOptions struct {
	URL           string
	APIToken      string
	MailboxID     int64
	CustomerEmail string
	AssignTo      int64
}

// NewFreeScoutClient returns a FreeScout client to use to make requests to the FreeScout external service.
func NewFreeScoutClient(opts *FreeScoutOptions) (*FreeScout, error) {
	if opts == nil {
		return nil, errors.New("missing FreeScout options")
	}
	if opts.URL == "" {
		return nil, errors.New("missing FreeScout URL")
	}
	parsedURL, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid FreeScout URL")
	}

	cleaned := *opts
	cleaned.URL = strings.TrimRight(opts.URL, "/")

	return &FreeScout{
		client: fleethttp.NewClient(),
		opts:   cleaned,
	}, nil
}

type freeScoutCustomer struct {
	Email string `json:"email"`
}

type freeScoutThread struct {
	Text     string             `json:"text"`
	Type     string             `json:"type"`
	Customer *freeScoutCustomer `json:"customer,omitempty"`
}

type freeScoutConversationPayload struct {
	Type      string             `json:"type"`
	MailboxID int64              `json:"mailboxId"`
	Subject   string             `json:"subject"`
	Customer  *freeScoutCustomer `json:"customer"`
	Threads   []freeScoutThread  `json:"threads"`
	Imported  bool               `json:"imported"`
	AssignTo  *int64             `json:"assignTo,omitempty"`
	Status    string             `json:"status,omitempty"`
}

// CreateFreeScoutConversation creates a conversation on the FreeScout server targeted by the FreeScout client.
// It returns the created conversation ID or an error.
func (f *FreeScout) CreateFreeScoutConversation(ctx context.Context, subject, message string) (int64, error) {
	payload := freeScoutConversationPayload{
		Type:      "email",
		MailboxID: f.opts.MailboxID,
		Subject:   subject,
		Customer: &freeScoutCustomer{
			Email: f.opts.CustomerEmail,
		},
		Threads: []freeScoutThread{
			{
				Text: message,
				Type: "customer",
				Customer: &freeScoutCustomer{
					Email: f.opts.CustomerEmail,
				},
			},
		},
		Imported: true,
		Status:   "active",
	}
	if f.opts.AssignTo > 0 {
		assignTo := f.opts.AssignTo
		payload.AssignTo = &assignTo
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	endpoint := fmt.Sprintf("%s/api/conversations", f.opts.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-FreeScout-API-Key", f.opts.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("freescout request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	resourceID := resp.Header.Get("Resource-ID")
	if resourceID == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return 0, nil
	}
	return id, nil
}

// FreeScoutConfigMatches returns true if the FreeScout client has been configured using those same options.
func (f *FreeScout) FreeScoutConfigMatches(opts *FreeScoutOptions) bool {
	return f.opts == *opts
}
