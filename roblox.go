package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

type UserPresence struct {
	UserPresenceType int       `json:"userPresenceType"`
	LastOnline       time.Time `json:"lastOnline"`
	PlaceID          int64     `json:"placeId"`
	RootPlaceID      int64     `json:"rootPlaceId"`
	GameID           string    `json:"gameId"`
	UniverseID       int64     `json:"universeId"`
	UserID           int64     `json:"userId"`
}

type UserPresenceResponse struct {
	UserPresences []UserPresence `json:"userPresences"`
}

type User struct {
	Description            string       `json:"description"`
	Created                string       `json:"created"`
	IsBanned               bool         `json:"isBanned"`
	ExternalAppDisplayName string       `json:"externalAppDisplayName"`
	HasVerifiedBadge       bool         `json:"hasVerifiedBadge"`
	ID                     int64        `json:"id"`
	Name                   string       `json:"name"`
	DisplayName            string       `json:"displayName"`
	Presence               UserPresence `json:"userPresence"`
	LastPresenceChange     time.Time    `json:"lastPresenceChange"`
	LastPresenceType       int          `json:"lastPresenceType"`
	Metrics                Metrics      `json:"metrics"`
}

func getHTTPClient() (*http.Client, error) {
	proxyHost := os.Getenv("PROXY_HOST")
	proxyPort := os.Getenv("PROXY_PORT")
	proxyUser := os.Getenv("PROXY_USER")
	proxyPassword := os.Getenv("PROXY_PASSWORD")

	// If no proxy is configured, return a default client
	if proxyHost == "" || proxyPort == "" {
		return &http.Client{}, nil
	}

	proxyAddr := fmt.Sprintf("%s:%s", proxyHost, proxyPort)

	// Create SOCKS5 dialer with optional authentication
	var auth *proxy.Auth
	if proxyUser != "" && proxyPassword != "" {
		auth = &proxy.Auth{
			User:     proxyUser,
			Password: proxyPassword,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	// Create HTTP transport with SOCKS5 dialer
	// Disable keep-alives to avoid EOF errors when proxy closes connections
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		DisableKeepAlives:   true,
		MaxIdleConns:        0,
		IdleConnTimeout:     0,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}

func getUsernameFromID(id int64) (User, error) {
	client, err := getHTTPClient()
	if err != nil {
		return User{}, err
	}

	// Retry logic for transient proxy errors
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Get(fmt.Sprintf("https://users.roblox.com/v1/users/%d", id))
		if err == nil {
			break
		}
		if attempt < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return User{}, err
	}

	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func checkPresence(userID int64) (UserPresence, error) {
	requestBody := struct {
		UserIDs []int64 `json:"userIds"`
	}{
		UserIDs: []int64{userID},
	}

	reqBytes, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error encoding request body:", err)
		return UserPresence{}, err
	}

	client, err := getHTTPClient()
	if err != nil {
		fmt.Println("Error getting HTTP client:", err)
		return UserPresence{}, err
	}

	// Retry logic for transient proxy errors
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Post("https://presence.roblox.com/v1/presence/users", "application/json", bytes.NewBuffer(reqBytes))
		if err == nil {
			break
		}
		if attempt < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}
	if err != nil {
		fmt.Println("Error making request:", err)
		return UserPresence{}, err
	}
	defer resp.Body.Close()

	var presenceResponse UserPresenceResponse
	err = json.NewDecoder(resp.Body).Decode(&presenceResponse)
	if err != nil {
		fmt.Println("Error decoding response:", err)
		return UserPresence{}, err
	}

	if len(presenceResponse.UserPresences) == 0 {
		fmt.Println("User not found")
		return UserPresence{}, nil
	}

	presence := presenceResponse.UserPresences[0]
	return presence, nil
}

func presenceTypeToString(presenceType int) string {
	switch presenceType {
	case 0:
		return "Offline"
	case 1:
		return "Online"
	case 2:
		return "InGame"
	case 3:
		return "InStudio"
	default:
		return "Unknown"
	}
}

func formatLastOnline(presence UserPresence) string {
	// If LastOnline is zero/null and user is active, they're currently active
	if presence.LastOnline.IsZero() || presence.LastOnline.Year() == 1 {
		if presence.UserPresenceType > 0 {
			return "currently active"
		}
		return "unknown"
	}

	minutesSinceLastOnline := int(time.Now().UTC().Sub(presence.LastOnline).Minutes())
	if minutesSinceLastOnline < 1 {
		return "just now"
	} else if minutesSinceLastOnline == 1 {
		return "1 minute ago"
	}
	return fmt.Sprintf("%d minutes ago", minutesSinceLastOnline)
}
