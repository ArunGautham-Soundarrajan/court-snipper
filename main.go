package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/config"
	"github.com/go-rod/rod"
)

func main() {

	// l := launcher.New().Headless(false).Devtools(true)
	// url := l.MustLaunch()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config")
	}

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(cfg.SignInURL)

	signIn(page, cfg)

	clearPopUps(page)

	checkAvailability(page, cfg)

	time.Sleep(5 * time.Hour)
}

func signIn(page *rod.Page, cfg *config.Config) {

	log.Println("Attempting to sign in")

	page.MustElement("#at-field-username_and_email").MustInput(cfg.UserName)
	page.MustElement("#at-field-password").MustInput(cfg.Password)

	// click the sign in button
	page.MustElement("#at-btn").MustClick()

	// check if it has log out to confirm sign-in
	if !hasSignedIn(page) {
		log.Fatal("Could not verify login.")
	}

}

func hasSignedIn(page *rod.Page) bool {
	_, err := page.Timeout(30*time.Second).ElementR("a", "/log out/i")

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Log Out button never appeared (30s timeout).")
		} else {
			log.Printf("Unknown error: %v\n", err)
		}
		return false
	}

	log.Println("Success: 'Log out' found. Proceeding...")
	return true
}

func clearPopUps(page *rod.Page) {

	cancelBtn, err := page.Timeout(30 * time.Second).Element("#onesignal-slidedown-cancel-button")

	if err == nil {
		log.Println("Popup detected: Clicking later")
		cancelBtn.MustClick()

		page.MustWaitIdle()
	} else {
		log.Println("No popups detected")
	}

}

func checkAvailability(page *rod.Page, cfg *config.Config) {

	if !hasSignedIn(page) {
		log.Fatal("Could not verify login.")
	}
	page.MustNavigate(cfg.CalendarURL)

}
