
FROM openjdk:8-jre
LABEL maintainer="zhblue0515@163.com"
RUN apt-get update && apt-get install -y tini
COPY dubbo-admin-distribution/target/dubbo-admin-0.4.0.jar /app.jar
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh

ENV JAVA_OPTS ""

ENTRYPOINT ["tini", "--", "/usr/local/bin/entrypoint.sh"]
EXPOSE 8080
