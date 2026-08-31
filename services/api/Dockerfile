FROM golang:1.26.6-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
USER app

EXPOSE 8080
CMD ["/app/api"]
