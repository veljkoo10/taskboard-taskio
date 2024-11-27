# Stage 1: Build the Angular app
FROM node:16 AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install --legacy-peer-deps  # Add the flag here to bypass the conflict
COPY . .
RUN npm run build --prod

# Stage 2: Serve the Angular app using a simple HTTP server
FROM httpd:alpine
COPY --from=build /app/dist/trello-ang/ /usr/local/apache2/htdocs/
EXPOSE 80
