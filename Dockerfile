FROM golang:1.12-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' \
        -o /bin/a.out ./cmd/cloudshell_open

FROM gcr.io/cloudshell-images/cloudshell:latest
RUN rm /google/devshell/bashrc.google.d/cloudshell_open.sh
COPY --from=build /bin/a.out /bin/cloudshell_open
