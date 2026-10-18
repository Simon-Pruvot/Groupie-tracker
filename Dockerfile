# syntax=docker/dockerfile:1
# Groupie Tracker (BeatFind) — serveur Go, bibliothèque standard uniquement.

FROM golang:1.25-alpine AS build
WORKDIR /app
COPY src ./src
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./src/main.go

FROM alpine:3.22
# ca-certificates : le serveur appelle l'API Groupie Tracker en HTTPS.
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H app
WORKDIR /app
COPY --from=build /out/server ./server
# Templates, CSS, images et musique sont lus relativement à /app (src/...).
COPY src ./src
USER app
ENV PORT=8080 \
    TZ=Europe/Paris
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO /dev/null "http://127.0.0.1:${PORT}/contact" || exit 1
CMD ["./server"]
