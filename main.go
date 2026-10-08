package main

import (
	"log"
	"time"

	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/bot"
	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/config"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const (
	maxBookingRetries = 3
	retryDelay        = 2 * time.Second
)

func main() {
	log.Println("Starting court booking bot...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}
	log.Println("Configuration loaded successfully")

	log.Println("Initializing browser (headless:", cfg.Headless, ")...")
	l := launcher.New().
		Headless(cfg.Headless).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage")
	// Use an installed Chromium (as in the container image) rather than
	// downloading one.
	if bin, ok := launcher.LookPath(); ok {
		l = l.Bin(bin)
	}
	browser := rod.New().ControlURL(l.MustLaunch()).MustConnect()
	defer browser.MustClose()
	log.Println("Browser connected")

	page := browser.MustPage(cfg.SignInURL)
	log.Println("Navigated to sign-in page")

	b, err := bot.NewBot(page, cfg)
	if err != nil {
		log.Fatal("Failed to create bot: ", err)
	}

	log.Println("Signing in...")
	if err := b.SignIn(); err != nil {
		log.Fatal("Sign-in failed: ", err)
	}
	log.Println("Sign-in successful")

	log.Println("Clearing popups...")
	b.ClearPopUps()

	times := cfg.BookingConfig.TimeSlots
	if len(times) == 0 {
		log.Fatal("No time slots configured in TIME_SLOTS")
	}
	log.Println("Checking availability for", len(times), "time slots:", times)

	successCount := 0
	for _, desiredTime := range times {
		log.Println("Checking availability for", desiredTime)
		element, err := b.CheckAvailability(desiredTime)
		if err != nil {
			log.Println("Error checking availability for", desiredTime, ":", err)
			continue
		}

		if element == nil {
			log.Println("No courts available for", desiredTime)
			continue
		}

		log.Println("Court found for", desiredTime, "- attempting to book")
		booked := false
		for attempt := 1; attempt <= maxBookingRetries; attempt++ {
			if err := b.BookCourt(element); err != nil {
				if attempt < maxBookingRetries {
					log.Println("Booking attempt", attempt, "failed for", desiredTime, "retrying in", retryDelay)
					time.Sleep(retryDelay)
				} else {
					log.Println("Error booking court for", desiredTime, "after", maxBookingRetries, "attempts:", err)
				}
			} else {
				log.Println("Successfully booked court for", desiredTime)
				booked = true
				successCount++
				break
			}
		}

		if booked {
			time.Sleep(5 * time.Second)
		}
	}

	log.Println("Booking attempt completed. Successfully booked:", successCount, "out of", len(times), "time slots")
}
