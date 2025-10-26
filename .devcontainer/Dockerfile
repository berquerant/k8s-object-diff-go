ARG GO_VERSION=1.25
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}

ENV DEVCONTAINER=true
RUN go install golang.org/x/tools/gopls@v0.20.0
RUN go install golang.org/x/tools/cmd/goimports@v0.38.0
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go build -o /usr/bin/golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint
