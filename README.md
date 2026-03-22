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
│   └── go.yml                        # CI/CD pipeline (manual + cron trigger)
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

### 4. GitHub Actions Pipeline (`.github/workflows/go.yml`)

**Triggers:**

- `workflow_dispatch` — Manual trigger via GitHub Actions UI
- `schedule` — Cron job (currently every Friday at 1:15 AM UTC)

**Critical Steps:**

1. **Install Dependencies:**

   ```bash
   sudo apt-get update
   sudo apt-get install -y chromium-browser xvfb
   ```

   - `xvfb` = X virtual framebuffer (required for headless browser display on Linux)

2. **Create `.env` from Secrets/Variables:**

   ```yaml
   echo "USER_NAME=${{ secrets.USER_NAME }}" >> .env
   echo "PASSWORD=${{ secrets.PASSWORD }}" >> .env
   echo "SIGN_IN_URL=${{ vars.SIGN_IN_URL }}" >> .env
   echo "CALENDAR_URL=${{ vars.CALENDAR_URL }}" >> .env
   echo "HEADLESS=${{ vars.HEADLESS }}" >> .env
   echo "TIME_SLOTS=${{ vars.TIME_SLOTS }}" >> .env
   ```

   - **Secrets** (encrypted): `USER_NAME`, `PASSWORD`
   - **Variables** (public): `SIGN_IN_URL`, `CALENDAR_URL`, `HEADLESS`, `TIME_SLOTS`

3. **Run with Virtual Display:**
   ```bash
   xvfb-run -a ./court-snipper
   DISPLAY=:99
   ```

   - `xvfb-run` provides virtual X11 display (required for CI/CD)

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

3. **GitHub Secrets Setup**
   - Must create secrets/variables in GitHub UI before running workflow
   - Secrets won't display in logs (for security)

4. **Virtual Display in CI/CD**
   - Headless still needs `xvfb` on Linux for display server
   - Don't forget `DISPLAY=:99` env variable

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

## Testing the Workflow

1. Push code to GitHub
2. Go to Actions tab → Select workflow
3. Click "Run workflow" button
4. Monitor logs in real-time
5. Check for:
   - Browser initialization success
   - Sign-in completion
   - Time slot checks
   - Booking success/failure

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

Update cron in `.github/workflows/go.yml`:

- `"0 8 * * 1"` — Every Monday at 8 AM UTC
- `"*/30 * * * *"` — Every 30 minutes
- `"0 0 * * *"` — Every day at midnight UTC

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
