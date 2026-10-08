package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ArunGautham-Soundarrajan/courtsnipper/internal/config"
	"github.com/go-rod/rod"
)

// CSS Selectors
const (
	usernameFieldSelector            = "#at-field-username_and_email"
	passwordFieldSelector            = "#at-field-password"
	signInButtonSelector             = "#at-btn"
	logoutButtonSelector             = "a"
	logoutButtonPattern              = "/log out/i"
	popupCancelButtonSelector        = "#onesignal-slidedown-cancel-button"
	timeSlotRowSelector              = "tr.minislot-row"
	courtGridSelector                = ".booking-centre-court-grid"
	availableCourtSelector           = "td.bg-info"
	bookButtonSelector               = "button"
	bookLinkPattern                  = "Book this court..."
	bookingConfirmationModalSelector = "#scheduleDialogue"
	confirmBookingButtonSelector     = "#confirmSchedule"
)

// Timeouts and delays
const (
	signInVerificationTimeout = 30 * time.Second
	popupCheckTimeout         = 30 * time.Second
	scrollCheckDelay          = 500 * time.Millisecond
	maxScrollAttempts         = 3
	scrollAmountPixels        = 800
)

type Bot struct {
	page                *rod.Page
	cfg                 *config.Config
	signInVerifyTimeout time.Duration
	popupCheckTimeout   time.Duration
}

// ErrNoAvailableCourt means the slot exists but every court is taken.
var ErrNoAvailableCourt = errors.New("no available court")

// BotError wraps bot-specific errors
type BotError struct {
	Operation string
	Message   string
	Err       error
}

func (e *BotError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("bot %s error: %s (%v)", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("bot %s error: %s", e.Operation, e.Message)
}

func (e *BotError) Unwrap() error { return e.Err }

func NewBotError(operation, message string, err error) *BotError {
	return &BotError{Operation: operation, Message: message, Err: err}
}

func NewBot(page *rod.Page, cfg *config.Config) (*Bot, error) {
	if page == nil {
		return nil, NewBotError("init", "page is nil", nil)
	}
	if cfg == nil {
		return nil, NewBotError("init", "config is nil", nil)
	}

	return &Bot{
		page:                page,
		cfg:                 cfg,
		signInVerifyTimeout: signInVerificationTimeout,
		popupCheckTimeout:   popupCheckTimeout,
	}, nil
}

func (b *Bot) SignIn() error {
	b.page.MustElement(usernameFieldSelector).MustInput(b.cfg.UserName)
	b.page.MustElement(passwordFieldSelector).MustInput(b.cfg.Password)
	b.page.MustElement(signInButtonSelector).MustClick()
	slog.Info("sign-in: credentials submitted")

	if ok, err := b.HasSignedIn(); !ok {
		if err != nil {
			return NewBotError("sign-in", "failed to verify login", err)
		}
		return NewBotError("sign-in", "logout button not found after sign-in", nil)
	}

	return nil
}

func (b *Bot) HasSignedIn() (bool, error) {
	_, err := b.page.Timeout(b.signInVerifyTimeout).ElementR(logoutButtonSelector, logoutButtonPattern)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (b *Bot) ClearPopUps() error {
	cancelBtn, err := b.page.Timeout(b.popupCheckTimeout).Element(popupCancelButtonSelector)

	if err == nil {
		cancelBtn.MustClick()
		b.page.MustWaitIdle()
		slog.Info("popup dismissed")
	}

	return nil
}

func (b *Bot) CheckAvailability(desiredTime string) (*rod.Element, error) {
	if desiredTime == "" {
		return nil, NewBotError("check-availability", "desired time cannot be empty", nil)
	}

	signedIn, err := b.HasSignedIn()
	if err != nil || !signedIn {
		return nil, NewBotError("check-availability", "sign-in verification failed", err)
	}

	wait := b.page.MustWaitNavigation()
	b.page.MustNavigate(b.cfg.CalendarURL)
	wait()
	slog.Info("calendar loaded", "slot", desiredTime)

	if _, err := b.page.Element(timeSlotRowSelector); err != nil {
		return nil, NewBotError("check-availability", "time slot row not found", err)
	}
	b.page.MustElement(timeSlotRowSelector).MustWaitVisible()

	container, err := b.page.Element(courtGridSelector)
	if err != nil {
		return nil, NewBotError("check-availability", "court grid not found", err)
	}

	if err := b.ScrollToTime(container, desiredTime); err != nil {
		return nil, NewBotError("check-availability", "failed to scroll to time", err)
	}

	row, err := b.page.ElementR(timeSlotRowSelector, desiredTime)
	if err != nil {
		return nil, NewBotError("check-availability", "time row not found after scroll", err)
	}

	row.MustScrollIntoView()

	firstAvailable, err := row.Element(availableCourtSelector)
	if err != nil || firstAvailable == nil {
		return nil, NewBotError("check-availability", fmt.Sprintf("no available courts for time %s", desiredTime), ErrNoAvailableCourt)
	}

	return firstAvailable, nil
}

func (b *Bot) ScrollToTime(container *rod.Element, targetTime string) error {
	if containerIsNil := container == nil; containerIsNil {
		return NewBotError("scroll-to-time", "container element is nil", nil)
	}

	if targetTime == "" {
		return NewBotError("scroll-to-time", "target time cannot be empty", nil)
	}

	for i := 0; i < maxScrollAttempts; i++ {
		row, err := b.page.ElementR(timeSlotRowSelector, targetTime)
		if err == nil && row != nil {
			row.MustScrollIntoView()
			return nil
		}

		container.MustEval(fmt.Sprintf(`() => this.scrollTop += %d`, scrollAmountPixels))
		b.page.MustWaitIdle()
		time.Sleep(scrollCheckDelay)
	}

	return NewBotError("scroll-to-time", fmt.Sprintf("could not find time row for %s after %d attempts", targetTime, maxScrollAttempts), nil)
}

func (b *Bot) BookCourt(courtElement *rod.Element) error {
	if courtElement == nil {
		return NewBotError("book-court", "court element is nil", nil)
	}

	bookBtn, err := courtElement.Element(bookButtonSelector)
	if err != nil {
		return NewBotError("book-court", "book button not found", err)
	}

	bookBtn.MustClick()

	bookLink, err := courtElement.ElementR("a", bookLinkPattern)
	if err != nil {
		return NewBotError("book-court", "booking link not found", err)
	}

	if err := bookLink.WaitVisible(); err != nil {
		return NewBotError("book-court", "booking link did not become visible", err)
	}

	bookLink.MustClick()
	slog.Info("booking: court selected, waiting for confirmation dialog")

	// Wait for confimation modal
	model, err := b.page.Timeout(15 * time.Second).Element(bookingConfirmationModalSelector)
	if err != nil {
		return NewBotError("book-court", "confirmation modal did not appear", err)
	}
	// #confirmSchedule
	button := model.MustElement(confirmBookingButtonSelector).MustWaitVisible()

	button.MustClick()
	slog.Info("booking: confirmed")

	return nil
}
