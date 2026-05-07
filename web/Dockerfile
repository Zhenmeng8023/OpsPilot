# syntax=docker/dockerfile:1

FROM node:24-alpine AS build

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web ./

ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run build

FROM nginx:1.29-alpine

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/web/dist /usr/share/nginx/html

EXPOSE 80
