# Court Snipper

A small Go bot that books sports courts on [ManageMyMatch](https://app.managemymatch.com) as soon as they become available. It signs in, opens the booking calendar, and books the first free court for each time slot you give it, in order.

It drives a real Chromium browser through [Rod](https://github.com/go-rod/rod) and is meant to run on a schedule, for example as a Kubernetes CronJob timed to when new slots open.

## How it works

1. Signs in with your account.
2. Dismisses any notification popup.
3. For each configured time slot:
   - opens the calendar page and scrolls to the slot;
   - if a court is free, books it, retrying up to three times;
   - if every court is taken, moves on.
4. Logs a summary. The process exits non-zero if signing in fails, or if any slot fails for a reason other than every court being taken.

## Configuration

All settings come from environment variables. When running locally you can put them in a `.env` file in the working directory instead.

| Variable | Description | Example |
|---|---|---|
| `USER_NAME` | Account username or email | `you@example.com` |
| `PASSWORD` | Account password | |
| `SIGN_IN_URL` | Sign-in page | `https://app.managemymatch.com/anon/signin` |
| `CALENDAR_URL` | Court calendar to book from. `day` is days from today. | `https://app.managemymatch.com/v2/courts?day=21` |
| `TIME_SLOTS` | Comma-separated start times, tried in order | `18:00, 19:00` |
| `HEADLESS` | Run Chromium without a window | `false` |

Never commit `.env`; it is already in `.gitignore`.

## Running locally

Requires Go 1.25+ and Chromium or Chrome. If neither is installed, Rod downloads a browser on first run.

```sh
go run .
```

Set `HEADLESS=false` to watch the browser while it works.

## Docker

The image bundles Chromium and runs the bot under `xvfb-run`, so `HEADLESS=false` works without a physical display.

```sh
docker build -t court-snipper .
docker run --rm --env-file .env court-snipper
```

Every push to `main` publishes `ghcr.io/arungautham-soundarrajan/court-snipper` with the tags `latest` and `sha-<commit>`.

## Logging

Logs are JSON lines on stdout, one per step. When something goes wrong the bot logs the error together with the page's URL and title at that moment, so a failed run shows where it stopped:

```json
{"level":"INFO","msg":"signing in"}
{"level":"ERROR","msg":"page at failure","url":"https://…/anon/signin","title":"Sign in"}
{"level":"ERROR","msg":"run failed","err":"signing in: bot sign-in error: logout button not found after sign-in"}
```

The whole run has an eight-minute limit, so a page element that never appears ends in a logged error rather than a hang.

## Project layout

```
main.go                     Entry point: browser setup, booking loop, logging
internal/bot/bot.go         Page automation: sign-in, availability, booking
internal/config/config.go   Loads settings from the environment or .env
Dockerfile                  Runtime image with Chromium and Xvfb
.github/workflows/          Publishes the image on push to main
```

## Note

This bot automates your own account. Use it in line with the booking site's terms, and keep the schedule reasonable.
