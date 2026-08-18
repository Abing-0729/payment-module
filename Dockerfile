FROM golang:1.24 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -o /out/commerce ./services/commerce/cmd/commerce

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/commerce /commerce
USER nonroot:nonroot
ENTRYPOINT ["/commerce"]