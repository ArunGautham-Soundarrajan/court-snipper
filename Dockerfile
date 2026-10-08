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
 && apt-get install -y --no-install-recommends chromium ca-certificates fonts-liberation xvfb xauth \
 && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --uid 10001 courtsnipper

COPY --from=build /out/court-snipper /usr/local/bin/

USER courtsnipper
WORKDIR /home/courtsnipper

CMD ["xvfb-run", "-a", "court-snipper"]
