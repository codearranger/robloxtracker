package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gregdel/pushover"
)

func notifyPresenceChange(user User) {
	app := pushover.New(os.Getenv("PUSHOVER_APP_TOKEN"))
	recipient := pushover.NewRecipient(os.Getenv("PUSHOVER_USER_KEY"))

	minutesSinceLastState := int(time.Now().UTC().Sub(user.LastPresenceChange).Minutes())
	lastOnlineStr := formatLastOnline(user.Presence)

	lastPresenceType := presenceTypeToString(user.LastPresenceType)
	presenceType := presenceTypeToString(user.Presence.UserPresenceType)
	message := pushover.NewMessage(fmt.Sprintf("User %s is now %s, was %s for %d minutes, last online: %s",
		user.Name, presenceType, lastPresenceType, minutesSinceLastState, lastOnlineStr))
	message.Title = "Roblox Presence Change"
	message.URL = fmt.Sprintf("https://www.roblox.com/users/%d/profile", user.ID)
	message.URLTitle = "View Profile"
	message.Sound = "magic"

	thumbnailData, err := downloadThumbnail(user.ID)
	if err != nil {
		log.Println(err)
	} else {
		message.AddAttachment(thumbnailData)
	}

	log.Printf("Presence state changed to %s. Notifying. \"%s\"\n", presenceTypeToString(user.Presence.UserPresenceType), message.Message)

	_, err = app.SendMessage(message, recipient)
	if err != nil {
		log.Println(err)
	}
}
