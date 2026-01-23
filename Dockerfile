FROM registry.access.redhat.com/ubi9/go-toolset:latest AS base

	WORKDIR /app

	ENV GOBIN=/opt/app-root/bin
	RUN go install github.com/air-verse/air@latest
	RUN go install github.com/air-verse/air@latest

	COPY go.mod go.sum ./




# live reloading, you'll have to mount the code
FROM base AS dev
	ENV PORT=8080
	ENV HTMX_SRC="/app/tmp/assets/htmx.min.js"
	ENV GOFLAGS="-mod=vendor"
	EXPOSE 8080

	CMD ["air", "-c", ".air.toml"]




FROM base AS builder
	ENV CGO_ENABLED=1
	COPY . .
	COPY vendor/ vendor/
	RUN go build -mod=vendor -o main ./cmd/web/main.go




FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS prod
	WORKDIR /app

	COPY --from=builder /app/main .
	COPY --from=builder /app/views ./views
	COPY tmp/assets/htmx.min.js /app/static/js/htmx.min.js
	COPY static ./static

	ENV HTMX_SRC="/app/static/js/htmx.min.js"
	ENV PORT=8080
	EXPOSE 8080

	USER 1001
	CMD ["./main"]
