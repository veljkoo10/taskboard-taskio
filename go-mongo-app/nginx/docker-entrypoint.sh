#!/bin/bash

# Use envsubst to replace environment variables in the template
envsubst '${NGINX_HTTP_PORT} ${NGINX_HTTPS_PORT} ${NGINX_SERVER_NAME} ${SSL_CERTIFICATE_PATH} ${SSL_CERTIFICATE_KEY_PATH} ${USER_SERVICE_HOST} ${USER_SERVICE_PORT} ${PROJECT_SERVICE_HOST} ${PROJECT_SERVICE_PORT} ${TASK_SERVICE_HOST} ${TASK_SERVICE_PORT} ${NOTIFICATION_SERVICE_HOST} ${NOTIFICATION_SERVICE_PORT}' \
    < /etc/nginx/templates/nginx.conf.template > /etc/nginx/nginx.conf

# Start NGINX
exec nginx -g "daemon off;"
