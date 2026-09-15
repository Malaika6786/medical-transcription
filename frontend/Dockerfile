# Production image for the Vue 3 frontend — builds the static bundle, then
# serves it from nginx. Added per SYSTMONE_INTEGRATION_REPORT.md, "No
# production IaC/container setup found".

FROM node:22-bookworm AS build
WORKDIR /src
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS run
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
