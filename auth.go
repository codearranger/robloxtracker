package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/anaminus/rbxauth"
)

const cookieFilename = "/data/cookies.txt"

func testAuth() {
	cfg := &rbxauth.Config{}

	cookies, err := loadCookiesFromFile()
	if err == nil {
		// If cookies are successfully loaded from file, no need to login again
		log.Println("Using cookies from file")
	} else {
		log.Println("No valid cookies found in file, logging in...")
		username := "your_username"
		password := []byte("your_password")

		cookies, step, err := cfg.Login(username, password)
		if err != nil {
			if step != nil {
				// Handle multi-step verification.
				log.Println("Two-step verification required.")
				// You can use step.Resend() and step.Verify() methods to handle the verification process.
			} else {
				log.Fatalf("Error logging in: %v", err)
			}
			return
		}

		err = saveCookiesToFile(cookies)
		if err != nil {
			log.Printf("Error saving cookies to file: %v", err)
		}
	}

	fmt.Println("Successfully logged in!")
	printCookies(cookies)
}

func loadCookiesFromFile() ([]*http.Cookie, error) {
	file, err := os.Open(cookieFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cookies, err := rbxauth.ReadCookies(file)
	if err != nil {
		return nil, err
	}

	return cookies, nil
}

func saveCookiesToFile(cookies []*http.Cookie) error {
	file, err := os.Create(cookieFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = rbxauth.WriteCookies(file, cookies)
	if err != nil {
		return err
	}

	return nil
}

func printCookies(cookies []*http.Cookie) {
	fmt.Println("Cookies:")
	for _, cookie := range cookies {
		fmt.Printf("%s: %s\n", cookie.Name, cookie.Value)
	}
}
