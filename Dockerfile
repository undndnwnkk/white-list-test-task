FROM golang:1.26-alpine AS source

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Integration tests: `docker compose run --rm tests` (see README).
FROM source AS test
ENV CGO_ENABLED=0
CMD ["go", "test", "-count=1", "./..."]

FROM source AS build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/app /app

EXPOSE 8080

ENTRYPOINT ["/app"]
