FROM debian:stable-slim AS builder

ARG NOTIFICATION_GRPC_PORT
ARG NOTIFICATION_HTTP_PORT

COPY create-env-properties.sh offloading/environment.properties ./
RUN bash create-env-properties.sh ./environment.properties

# Download and unzip the admin app
RUN apt update && apt install -y curl unzip
COPY download-unzip-nexus-artifact.sh ./
RUN bash download-unzip-nexus-artifact.sh ${NEXUS_URL} offloading /opt/app/lib


FROM amazoncorretto:11.0.21-al2023

RUN yum -y update && yum install -y unzip && yum install -y findutils

WORKDIR /opt/app

RUN mkdir logs
COPY --from=builder ./environment.properties /opt/app/* ./
COPY jar-entrypoint.sh entrypoint.sh
COPY redeploy.sh ./
RUN chmod +x entrypoint.sh redeploy.sh
ENTRYPOINT ["/opt/app/entrypoint.sh", "offloading", "com.jhi.funddataservices.offloading.config.MaestroOffloadingApplication"]
