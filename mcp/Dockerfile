FROM node:22-slim AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --production=false

COPY tsconfig.json ./
COPY src ./src
RUN npx tsc

RUN npm prune --production

FROM node:22-slim

WORKDIR /app

COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json ./

RUN mkdir -p /tmp/hf-cache && chown node:node /tmp/hf-cache

USER node

ENV HF_HOME=/tmp/hf-cache

EXPOSE 3000

CMD ["node", "dist/index.js"]
