package main

import (
	"log"
	"time"

	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/bot"
	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/config"
	"github.com/go-rod/rod"
)

const (
	maxBookingRetries = 3
	retryDelay        = 2 * time.Second
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(cfg.SignInURL)

	b, err := bot.NewBot(page, cfg)
	if err != nil {
		log.Fatal("Failed to create bot: ", err)
	}

	if err := b.SignIn(); err != nil {
		log.Fatal("Sign-in failed: ", err)
	}

	b.ClearPopUps()

	times := []string{"18:00", "19:00"}

	for _, desiredTime := range times {
		element, err := b.CheckAvailability(desiredTime)
		if err != nil {
			log.Println("Error checking availability for", desiredTime, ":", err)
			continue
		}

		if element == nil {
			continue
		}

		booked := false
		for attempt := 1; attempt <= maxBookingRetries; attempt++ {
			if err := b.BookCourt(element); err != nil {
				if attempt < maxBookingRetries {
					log.Println("Booking attempt", attempt, "failed for", desiredTime, "retrying...")
					time.Sleep(retryDelay)
				} else {
					log.Println("Error booking court for", desiredTime, "after", maxBookingRetries, "attempts:", err)
				}
			} else {
				booked = true
				break
			}
		}

		if booked {
			time.Sleep(5 * time.Second)
		}
	}
}
