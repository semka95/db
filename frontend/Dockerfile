# this runs a lil nginx instance to serve the app
FROM nginx:alpine
EXPOSE 3000
COPY /build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
