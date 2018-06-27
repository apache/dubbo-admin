# Start with a base image containing Java runtime
FROM openjdk:8-jdk-alpine

# Add Maintainer Info
LABEL maintainer="x.meng@outlook.com"

# Add a volume pointing to /tmp
VOLUME /tmp

# Make port 8080 available to the world outside this container
EXPOSE 7001

# The application's jar file
ARG JAR_FILE=target/dubbo-admin-2.0.0.jar

# Add the application's jar to the container
ADD ${JAR_FILE} dubbo-admin.jar

# timezone
ARG TIME_ZONE=Asia/Shanghai

RUN apk add -U tzdata \
    && cp  /usr/share/zoneinfo/${TIME_ZONE} /etc/localtime

ENV ZOOKEEPER="127.0.0.1:2181"
ENV ROOT_PASS="root"
ENV GUEST_PASS="guest"

# Run the jar file
#ENTRYPOINT ["java","-Djava.security.egd=file:/dev/./urandom", "$ZOOKEEPER", "$ROOT_PASS", "$GUEST_PASS","-jar","/dubbo-admin.jar"]
ENTRYPOINT java -Djava.security.egd=file:/dev/./urandom -Ddubbo.registry.address=zookeeper://$ZOOKEEPER  -Ddubbo.admin.root.password=$ROOT_PASS -Ddubbo.admin.guest.password=$GUEST_PASS -jar /dubbo-admin.jar