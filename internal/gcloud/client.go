package gcloud

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client talks to the Google APIs the app needs, as the logged-in user.
type Client struct {
	calendar *calendar.Service
	gmail    *gmail.Service
}

// NewClient loads the OAuth client config from credentialsPath, logs in
// (see GetToken) and sets up the Calendar and Gmail services.
// The login flow reads from stdin, so call this before starting the UI.
func NewClient(ctx context.Context, credentialsPath string) (*Client, error) {
	// Load the OAuth client config downloaded from Google Cloud.
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", credentialsPath, err)
	}
	// Read the calendar (but not change it) and send email as the user.
	// If you change these scopes, delete token.json so the login flow asks for them again.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	// config.Client returns an *http.Client that adds auth and auto-refreshes the token.
	httpClient := config.Client(ctx, GetToken(config))

	calSrv, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("calendar service: %w", err)
	}
	gmailSrv, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("gmail service: %w", err)
	}
	return &Client{calendar: calSrv, gmail: gmailSrv}, nil
}
