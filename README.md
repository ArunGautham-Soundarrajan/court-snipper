# Court Snipper Bot

Automated court booking bot that uses headless browser automation to book tennis/sports courts at configured times.

## Project Structure

```
.
├── main.go                           # Entry point - orchestration & booking loop
├── internal/
│   ├── bot/
│   │   └── bot.go                   # Browser automation & booking logic
│   └── config/
│       └── config.go                # Configuration loading from .env
├── .github/workflows/
│   └── publish.yml                   # Builds and publishes the container image
└── .env                             # Configuration file (secrets excluded)
```

## Key Components

### 1. Configuration System (`internal/config/config.go`)

Configuration is loaded from `.env` file and parsed into a `Config` struct:

```env
USER_NAME=your_username
PASSWORD=your_password
SIGN_IN_URL=https://booking-site.com/signin
CALENDAR_URL=https://booking-site.com/calendar
HEADLESS=true
TIME_SLOTS=18:00, 19:00, 20:00
```

**Important Notes:**

- `TIME_SLOTS` is comma-separated and automatically parsed into a slice
- Spaces around commas are trimmed automatically
- `HEADLESS=true/false` controls whether browser runs headless or headed mode

### 2. Bot Logic (`internal/bot/bot.go`)

Core booking methods:

- **`SignIn()`** — Enters credentials and verifies successful login
- **`ClearPopUps()`** — Dismisses any popup modals
- **`CheckAvailability(time)`** — Navigates to calendar, scrolls to desired time, returns court element if available
- **`BookCourt(element)`** — Clicks booking button and verifies confirmation

**Common Debugging Issues:**

#### Selector Problems

The biggest gotcha: **distinguish between CSS classes (`.`) and IDs (`#`)**

```go
// WRONG - looking for class .scheduleDialogue
model.MustElement("div.scheduleDialogue")

// CORRECT - looking for id #scheduleDialogue
model.MustElement("div#scheduleDialogue")
```

Use browser DevTools to inspect the HTML and check if it's a `class` or `id` attribute.

#### Element Not Found

- Increase timeout: `.Timeout(30 * time.Second).Element(...)`
- Verify element exists: `element.MustWaitVisible()`
- Check the page actually navigated: Add logging before `.Element()` call

#### Button Click Not Registering

- Ensure button is visible: `button.MustScrollIntoView()`
- Wait for modal to appear: Use `.Timeout()` with reasonable delay
- Some buttons need JavaScript focus: `button.MustFocus()` before `.MustClick()`

### 3. Main Orchestration (`main.go`)

**Booking Flow:**

1. Load configuration from `.env`
2. Initialize browser with launcher flags (headless, sandbox, GPU, dev-shm)
3. Sign in to booking site
4. Clear any popups
5. **For each time slot:**
   - Check availability
   - If available, attempt to book with retry logic (3 attempts, 2-second backoff)
   - If successful, wait 5 seconds before next slot
6. Print summary

**Retry Logic:**

- `maxBookingRetries = 3` attempts per slot
- `retryDelay = 2 * time.Second` between attempts
- Exponential backoff logging shows attempt number

### 4. Deployment

The bot runs as a Kubernetes CronJob in the homelab cluster (Fridays at 01:15 Europe/London); the manifests live in the `homelab-cluster` repo under `clusters/config/court-snipper`.

- `.github/workflows/publish.yml` builds the `Dockerfile` and pushes `ghcr.io/arungautham-soundarrajan/court-snipper:latest` (and `sha-<commit>`) on every push to `main`. The next scheduled run picks it up.
- The image runs `xvfb-run -a court-snipper`, so Chromium has a virtual display even with `HEADLESS=false`.
- Configuration comes from environment variables (no `.env` in the container). `USER_NAME`, `PASSWORD`, `SIGN_IN_URL` and `CALENDAR_URL` are synced from Infisical (`/court-snipper`); `HEADLESS` and `TIME_SLOTS` are set in the CronJob.

## Browser Launcher Flags

These flags are essential for CI/CD environments:

```go
launcher.New().
    Headless(cfg.Headless).
    Set("no-sandbox").                 // Permissions in container
    Set("disable-gpu").               // Prevent GPU acceleration issues
    Set("disable-dev-shm-usage")      // Prevent shared memory issues
```

**Common Errors:**

- "No usable sandbox" → Missing `--no-sandbox` flag
- "Missing X server" → Missing `xvfb` and `DISPLAY` env variable
- "GPU crash" → Missing `--disable-gpu` flag

## Development Workflow

### Local Testing

```bash
# Create .env file locally
cp .env.example .env
# Edit with your credentials

# Run locally (headed mode for debugging)
go run main.go
```

### Debugging Navigation Issues

1. Set `HEADLESS=false` in `.env` to see browser window
2. Add logging before problematic selectors
3. Use `Eval()` to inspect element content:
   ```go
   text, _ := element.Eval(`el => el.textContent`)
   log.Println("Element contains:", text)
   ```

### Highlighting Elements (for verification)

```go
button := element.MustElement("#confirmSchedule")
button.MustScrollIntoView()
button.Eval(`el => { el.style.border = '3px solid red !important'; }`)
time.Sleep(3 * time.Second)  // Pause to see it
button.MustClick()
```

### Taking Screenshots

```go
screenshot, _ := b.page.Screenshot(false, &proto.PageCaptureScreenshot{Format: proto.ImageFormatPng})
// Save or log screenshot for debugging
```

## Common Things to Forget

1. **Element Selectors**
   - Always check if it's `.class` or `#id` in the HTML
   - Update selectors if website HTML changes

2. **Timeouts**
   - Increase timeout if page is slow to load
   - Default timeouts are often too short for real websites

3. **Secrets**
   - Credentials and URLs live in Infisical at `/court-snipper`, never in Git

4. **Virtual Display**
   - Headed Chromium on Linux needs a display server; the image runs the bot under `xvfb-run`

5. **Configuration Format**
   - TIME_SLOTS needs commas as separator: `18:00, 19:00`
   - Extra spaces are fine and will be trimmed

6. **Retry Logic**
   - Retry only happens if booking fails
   - Each slot is independent (one slot failing doesn't skip others)

7. **Browser Cleanup**
   - Always `defer browser.MustClose()` to prevent zombie processes
   - Important in CI/CD to avoid resource leaks

8. **Modal/Popup Handling**
   - Popups can appear at different times
   - `ClearPopUps()` is safe to call multiple times
   - If popup blocking booking, increase wait time before checking for button

## Extending the Bot

### Add New Booking Sites

- Update CSS selectors in bot.go for the new site
- May need new methods if workflow is different
- Test locally first with `HEADLESS=false`

### Add Notifications

- On successful booking: Send email/Slack message
- Integrate with notification service after `BookCourt()` succeeds

### Add Logging/Monitoring

- Log to file instead of stdout: `logfile, _ := os.Create("bookings.log")`
- Send logs to monitoring service (CloudWatch, DataDog, etc.)

### Change Schedule

Edit `schedule` in `clusters/config/court-snipper/cronjob.yaml` in the `homelab-cluster` repo.

## Error Handling

All bot methods return `error` via custom `BotError` type:

```go
type BotError struct {
    Operation string // Which operation failed (e.g., "book-court")
    Message   string // What went wrong
    Err       error  // Underlying error
}
```

Errors are logged in main.go but don't stop the program—allows checking all time slots even if one fails.
