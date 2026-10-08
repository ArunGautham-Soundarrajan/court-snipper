package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/bot"
	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/config"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	maxBookingRetries = 3
	retryDelay        = 2 * time.Second
	// runTimeout caps all page work, so a selector that never appears fails
	// with a logged error instead of hanging until the job is killed.
	runTimeout = 8 * time.Minute
	// pageInfoTimeout bounds the page read made after a failure.
	pageInfoTimeout = 10 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if len(cfg.TimeSlots) == 0 {
		return errors.New("no time slots configured in TIME_SLOTS")
	}
	slog.Info("config loaded", "headless", cfg.Headless, "time_slots", cfg.TimeSlots)

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
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("launching browser: %w", err)
	}
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("connecting to browser: %w", err)
	}
	defer browser.Close()
	slog.Info("browser started")

	page, err := browser.Page(proto.TargetCreateTarget{URL: cfg.SignInURL})
	if err != nil {
		return fmt.Errorf("opening sign-in page: %w", err)
	}

	return book(page.Timeout(runTimeout), cfg)
}

// book signs in and tries each configured slot in turn. Once every slot has
// been tried, it returns an error if any slot failed for a reason other than
// having no free court.
func book(page *rod.Page, cfg *config.Config) (err error) {
	// The bot uses rod's Must* helpers, which panic. Turn that into an error,
	// and record where the browser was when it happened.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			logPageState(page)
		}
	}()

	b, err := bot.NewBot(page, cfg)
	if err != nil {
		return err
	}

	slog.Info("signing in")
	if err := b.SignIn(); err != nil {
		logPageState(page)
		return fmt.Errorf("signing in: %w", err)
	}
	slog.Info("signed in")

	b.ClearPopUps()

	booked, failed := 0, 0
	for _, slot := range cfg.TimeSlots {
		log := slog.With("slot", slot)

		log.Info("checking availability")
		court, err := b.CheckAvailability(slot)
		if errors.Is(err, bot.ErrNoAvailableCourt) {
			log.Info("no courts available")
			continue
		}
		if err != nil {
			log.Error("checking availability failed", "err", err)
			logPageState(page)
			failed++
			continue
		}

		log.Info("court available, booking")
		if err := bookWithRetries(b, court, log); err != nil {
			log.Error("booking failed", "attempts", maxBookingRetries, "err", err)
			logPageState(page)
			failed++
			continue
		}
		log.Info("court booked")
		booked++
		time.Sleep(5 * time.Second)
	}

	slog.Info("run complete", "booked", booked, "failed", failed, "slots", len(cfg.TimeSlots))
	if failed > 0 {
		return fmt.Errorf("%d of %d slots failed", failed, len(cfg.TimeSlots))
	}
	return nil
}

func bookWithRetries(b *bot.Bot, court *rod.Element, log *slog.Logger) error {
	var err error
	for attempt := 1; attempt <= maxBookingRetries; attempt++ {
		if err = b.BookCourt(court); err == nil {
			return nil
		}
		if attempt < maxBookingRetries {
			log.Warn("booking attempt failed, retrying", "attempt", attempt, "retry_in", retryDelay.String(), "err", err)
			time.Sleep(retryDelay)
		}
	}
	return err
}

// logPageState logs the page's URL and title, to show where a failure
// happened. It is best effort and uses its own deadline.
func logPageState(page *rod.Page) {
	info, err := page.Context(context.Background()).Timeout(pageInfoTimeout).Info()
	if err != nil {
		slog.Warn("could not read page state", "err", err)
		return
	}
	slog.Error("page at failure", "url", info.URL, "title", info.Title)
}
