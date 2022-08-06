# Start from the latest golang base image
FROM golang:1.18-alpine3.15 as builder

RUN apk add --no-cache git build-base linux-headers

WORKDIR /deliverbot

# download and cache go mod
COPY ./go.* .
RUN go env -w GO111MODULE=on && go mod download

COPY . .

ENV GOCACHE /root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 make

######## Start a new stage #######
FROM alpine:3.15
LABEL maintainer="none<none.one>"

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /deliverbot/deliverbot /bin/deliverbot

EXPOSE 8000
WORKDIR /
ENTRYPOINT ["/bin/deliverbot"]