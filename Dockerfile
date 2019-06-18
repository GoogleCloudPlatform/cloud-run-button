FROM golang:1.12-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' \
        -o /bin/a.out ./cmd/cloudshell_open
WORKDIR /tmp
RUN wget https://github.com/buildpack/pack/releases/download/v0.2.1/pack-v0.2.1-linux.tgz && tar -xzf pack-v0.2.1-linux.tgz
RUN wget https://raw.githubusercontent.com/buildpack/pack/v0.2.1/LICENSE

FROM gcr.io/cloudshell-images/cloudshell:latest
RUN rm /google/devshell/bashrc.google.d/cloudshell_open.sh
COPY --from=build /bin/a.out /bin/cloudshell_open
COPY --from=build /tmp/pack /opt/pack/pack
COPY --from=build /tmp/LICENSE /opt/pack/LICENSE
RUN ln -s /opt/pack/pack /usr/local/bin/pack
