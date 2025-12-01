package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// Get the metrics
	metrics := robloxMetrics(reg)

	// Check for required environment variables
	userIDsStr := os.Getenv("ROBLOX_USER_IDS")
	userIDsStrSlice := strings.Split(userIDsStr, ",")
	userIDs := make([]int64, len(userIDsStrSlice))

	for i, idStr := range userIDsStrSlice {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Println(err)
			return
		}
		userIDs[i] = id
	}

	notifyIDsStr := os.Getenv("NOTIFY_ROBLOX_USER_IDS")
	notifyIDsStrSlice := strings.Split(notifyIDsStr, ",")
	notifyIDs := make([]int64, len(notifyIDsStrSlice))

	for i, idStr := range notifyIDsStrSlice {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Println(err)
			return
		}
		notifyIDs[i] = id
	}

	for _, userID := range userIDs {
		go monitorUser(reg, userID, metrics, notifyIDs)
		time.Sleep(time.Second * 1) // Rate limit requests to the API by staggering the requests
	}

	// Expose metrics and custom registry via an HTTP server
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func monitorUser(reg *prometheus.Registry, userID int64, metrics *Metrics, notifyIDs []int64) {
	// Get the username string
	user, err := getUsernameFromID(userID)
	if err != nil {
		log.Println(err)
		return
	}

	user.LastPresenceChange = time.Now().UTC()

	// Start presence checker
	presenceState := 0
	user.LastPresenceType = presenceState
	t := time.NewTicker(time.Second * 5)

	// Check presence every 5 seconds
	for range t.C {
		// Check presence
		user.Presence, err = checkPresence(user.ID)
		if err != nil {
			log.Println(err)
			return
		}

		// Check if presence has changed and notify
		if presenceState != user.Presence.UserPresenceType {
			// Update last online time
			user.LastPresenceType = presenceState

			// Log presence change
			log.Printf("User %s is %s, last online: %s\n", user.Name, presenceTypeToString(user.Presence.UserPresenceType), formatLastOnline(user.Presence))

			log.Printf("Presence: %#v\n", user.Presence)

			// Check if the user is in the list of IDs that we want to receive notifications for
			for _, id := range notifyIDs {
				if user.ID == id {
					// Notify if user is online
					notifyPresenceChange(user)
					user.LastPresenceChange = time.Now().UTC()
					break
				}
			}
		}

		// Update metrics
		metrics.UserPresenceType.With(prometheus.Labels{"userid": user.Name}).Set(float64(user.Presence.UserPresenceType))

		// Update presence state
		presenceState = user.Presence.UserPresenceType
	}
}
