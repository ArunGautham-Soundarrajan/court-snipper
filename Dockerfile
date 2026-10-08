FROM golang:1.26-trixie AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/court-snipper .

FROM debian:trixie-slim

# main launches /usr/bin/chromium via launcher.LookPath. xvfb-run gives it a
# virtual display, so HEADLESS=false works as it did on GitHub Actions.
RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends chromium ca-certificates fonts-liberation xvfb xauth tini \
 && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --uid 10001 courtsnipper

COPY --from=build /out/court-snipper /usr/local/bin/

USER courtsnipper
WORKDIR /home/courtsnipper

# xvfb-run hangs as PID 1: it waits for Xvfb's ready signal, which never
# arrives. tini runs as PID 1 instead and also reaps Chromium's child processes.
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["xvfb-run", "-a", "court-snipper"]
