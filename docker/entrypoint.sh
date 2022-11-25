#!/bin/bash
set -e
exec java $JAVA_OPTS $REGISTRY_ADDRESS $CONFIG_CENTER_ADDRESS $META_ADDRESS -Djava.security.egd=file:/dev/./urandom -jar /app.jar
