# -- BUILD --
FROM node:14.17-alpine as build

WORKDIR /app/ui/src


COPY . .

RUN npm i
RUN npm run build

# -- RELEASE --
FROM nginx:stable-alpine as release

COPY --from=build /app/ui/src/target/dist /usr/share/nginx/html

WORKDIR /usr/share/nginx/html
EXPOSE 8080

CMD ["/bin/sh", "-c", "nginx -g \"daemon off;\""]
