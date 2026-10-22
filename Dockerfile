# ========================================================
# Stage: Builder
# ========================================================
FROM golang:1.27.2-alpine AS builder
WORKDIR /app
ARG TARGETARCH
ARG GOPROXY=https://proxy.golang.org,direct
ARG ALPINE_MIRROR=dl-cdn.alpinelinux.org

RUN sed -i "s#dl-cdn.alpinelinux.org#${ALPINE_MIRROR}#g" /etc/apk/repositories \
    && apk --no-cache --update add \
    build-base \
    gcc \
    curl \
    unzip

ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
ENV GOPROXY=$GOPROXY

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=superxray-go-build-${TARGETARCH},target=/root/.cache/go-build,sharing=locked \
    go build -mod=readonly -ldflags "-w -s" -o build/x-ui main.go
RUN chmod +x DockerInit.sh && ./DockerInit.sh "$TARGETARCH"

# ========================================================
# Stage: Final Image of SuperXray
# ========================================================
FROM alpine:3.22
ENV TZ=Asia/Tehran
WORKDIR /app
ARG ALPINE_MIRROR=dl-cdn.alpinelinux.org

RUN sed -i "s#dl-cdn.alpinelinux.org#${ALPINE_MIRROR}#g" /etc/apk/repositories \
    && apk add --no-cache --update \
    ca-certificates \
    tzdata \
    fail2ban \
    bash \
    curl \
    openssl

COPY --from=builder /app/build/ /app/
COPY --from=builder /app/DockerEntrypoint.sh /app/
COPY --from=builder /app/x-ui.sh /usr/bin/x-ui


# Configure fail2ban
RUN rm -f /etc/fail2ban/jail.d/alpine-ssh.conf \
  && cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local \
  && sed -i "s/^\[ssh\]$/&\nenabled = false/" /etc/fail2ban/jail.local \
  && sed -i "s/^\[sshd\]$/&\nenabled = false/" /etc/fail2ban/jail.local \
  && sed -i "s/#allowipv6 = auto/allowipv6 = auto/g" /etc/fail2ban/fail2ban.conf

RUN chmod +x \
  /app/DockerEntrypoint.sh \
  /app/x-ui \
  /usr/bin/x-ui

ENV XUI_ENABLE_FAIL2BAN="true"
EXPOSE 2053
VOLUME [ "/etc/x-ui" ]
CMD [ "./x-ui" ]
ENTRYPOINT [ "/app/DockerEntrypoint.sh" ]
