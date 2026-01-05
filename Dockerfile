FROM ghcr.io/gohugoio/hugo:v0.152.2 AS builder

WORKDIR /src
COPY . .

USER root

ARG BASE_URL
ENV HUGO_PARAMS_LOGO_IMAGE="/images/logokin.png"
ENV HUGO_PARAMS_LOGO_LINK="/"

RUN hugo --minify -b "${BASE_URL}"

FROM nginx:alpine3.22-slim

ARG GIT_SHA
ARG VERSION

LABEL org.opencontainers.image.title="kinho-blog" \
      org.opencontainers.image.description="Kin Hong NG's Blog Website" \
      org.opencontainers.image.source="https://github.com/k1nho/k1nho.github.io" \
      org.opencontainers.image.licenses="Apache-2.0"

LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${GIT_SHA}

COPY --from=builder /src/public /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
