
FROM golang:1.24-alpine3.22 AS gobuilder
WORKDIR /app

RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

ENV USER=appuser
ENV UID=10001 
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "${UID}" \    
    "${USER}"
    
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w" -o ./sharp-cred-manager ./cmd/api/main.go

FROM scratch AS runner
COPY --from=gobuilder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuilder /etc/passwd /etc/passwd
COPY --from=gobuilder /etc/group /etc/group
WORKDIR /app
COPY --from=gobuilder /app/sharp-cred-manager .
COPY --from=gobuilder /app/frontend ./frontend
COPY --from=gobuilder /app/public ./public
USER appuser:appuser
EXPOSE 8000
ENTRYPOINT ["./sharp-cred-manager"]
