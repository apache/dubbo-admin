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

# -- BUILD --
FROM node:14.17-alpine as build

WORKDIR /app/ui/src


COPY . .

RUN npm i
RUN npm run build

# -- RELEASE --
FROM nginx:stable-alpine as release

COPY --from=build /app/ui/src/dist /usr/share/nginx/html

WORKDIR /usr/share/nginx/html
EXPOSE 80

CMD ["/bin/sh", "-c", "nginx -g \"daemon off;\""]
