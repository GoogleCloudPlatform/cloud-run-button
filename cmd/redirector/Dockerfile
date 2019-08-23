FROM golang:1-alpine AS build
RUN apk add --no-cache git
WORKDIR /src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server -ldflags="-w -s" . # rundev

FROM scratch
COPY --from=build /server /server
ENTRYPOINT ["/server"]
