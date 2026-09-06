FROM node:22-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.25-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /study ./cmd/study

FROM alpine:3.22
RUN addgroup -g 10001 study && adduser -D -u 10001 -G study study && mkdir /data && chown study:study /data
COPY --from=backend /study /usr/local/bin/study
USER 10001:10001
ENV STUDY_ADDR=:8080 STUDY_DB=/data/study.db
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/study"]
