# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


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