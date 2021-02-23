#!/bin/bash
set -e

exec java $JAVA_OPTS -Djava.security.egd=file:/dev/./urandom -jar /app.jar
