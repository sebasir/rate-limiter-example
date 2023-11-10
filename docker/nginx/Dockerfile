FROM nginx:latest

ARG RATE_LIMITER_HOST
ARG RATE_LIMITER_HTTP_PORT
ARG NOTIFICATION_HOST
ARG NOTIFICATION_HTTP_PORT

WORKDIR /etc/nginx

RUN rm conf.d/*

COPY /docker/nginx/nginx.conf.template conf.d/nginx.conf

WORKDIR /etc/nginx/conf.d/

RUN sed -i "s/{{RATE_LIMITER_HOST}}/$RATE_LIMITER_HOST/g" nginx.conf
RUN sed -i "s/{{RATE_LIMITER_HTTP_PORT}}/$RATE_LIMITER_HTTP_PORT/g" nginx.conf
RUN sed -i "s/{{NOTIFICATION_HOST}}/$NOTIFICATION_HOST/g" nginx.conf
RUN sed -i "s/{{NOTIFICATION_HTTP_PORT}}/$NOTIFICATION_HTTP_PORT/g" nginx.conf
RUN find . -type f | xargs chmod 0444 && find . -type d | xargs chmod 0555
