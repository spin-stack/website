# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
# The site's Node toolchain, so a host needs Go and Docker and nothing of Node's: the install, the
# build and the dev server run here, at the Node the image names. Go stays on the host, where
# refgen builds spin's CLI and every check but the build runs. The images are versions.yaml's,
# handed in by the Taskfile (versions args).
ARG GOLANG_IMAGE
ARG NODE_IMAGE

# go-tools' sfw, at the commit go.mod pins: the install goes through Socket Firewall in the image
# as it did on the host, and sfw check holds this file to it.
FROM ${GOLANG_IMAGE} AS sfw
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,id=spinbox-docs-gomod,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -o /sfw github.com/spin-stack/go-tools/cmd/sfw

FROM ${NODE_IMAGE} AS deps
# git: the build reads each page's last commit. COREPACK_HOME outside a home: task dev runs as the
# host's user, who finds pnpm where root put it.
ENV CI=true COREPACK_ENABLE_DOWNLOAD_PROMPT=0 COREPACK_HOME=/usr/local/share/corepack
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git \
 && rm -rf /var/lib/apt/lists/*
COPY --from=sfw /sfw /usr/local/bin/sfw
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
# pnpm itself comes from corepack, at the version package.json names: the firewall stands between
# a package manager and the registry, not in front of the package manager. No store cache: a
# package restored from one never reaches the registry, so the firewall would not see it. The
# firewall's own binary is cached, checked against its pin on every run. GITHUB_TOKEN, when the
# host has one, is what sfw downloads that binary with.
RUN corepack enable pnpm && corepack install \
 && chmod -R a+rX "$COREPACK_HOME"
RUN --mount=type=cache,id=spinbox-docs-sfw,target=/root/.cache/spin-stack/sfw \
    --mount=type=secret,id=github_token,env=GITHUB_TOKEN \
    sfw pnpm install --frozen-lockfile \
 # task dev runs as the host's user: Vite and Astro write their caches under node_modules.
 && chmod a+rwx node_modules

FROM deps AS build
ARG SITE_URL
ARG BASE_PATH
ENV SITE_URL=$SITE_URL BASE_PATH=$BASE_PATH
COPY . .
# The few rules the build keeps in JavaScript, with node's own runner: no dependency.
RUN node --test 'src/lib/*.test.mjs' && pnpm build

FROM scratch AS dist
COPY --from=build /app/dist/ /
