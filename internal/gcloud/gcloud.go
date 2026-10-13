package gcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2"
)

// getToken loads token.json if it exists, otherwise runs the login flow and saves it.
func GetToken(config *oauth2.Config) *oauth2.Token {
	if f, err := os.Open("token.json"); err == nil {
		defer f.Close()
		tok := &oauth2.Token{}
		if json.NewDecoder(f).Decode(tok) == nil {
			return tok
		}
	}

	config.RedirectURL = "http://localhost"
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("Open this URL, approve access, then copy the `code=` value\nfrom the (broken) localhost page's address bar and paste it here:\n\n%s\n\nCode: ", url)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("read code: %v", err)
	}
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("exchange code: %v", err)
	}

	f, err := os.OpenFile("token.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("save token: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(tok); err != nil {
		log.Fatalf("save token: %v", err)
	}
	return tok
}
