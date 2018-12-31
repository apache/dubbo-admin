/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.apache.dubbo.admin.service;

import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;

import java.util.List;

/**
 * Query service for consumer info
 *
 */
public interface ConsumerService {

    List<Consumer> findByService(String serviceName);

    Consumer findConsumer(String id);

    String getConsumerMetadata(MetadataIdentifier consumerIdentifier);

    List<Consumer> findAll();

    /**
     * query for all consumer addresses
     */
    List<String> findAddresses();

    List<String> findAddressesByApplication(String application);

    List<String> findAddressesByService(String serviceName);

    List<Consumer> findByAddress(String consumerAddress);

    List<String> findServicesByAddress(String consumerAddress);

    List<String> findApplications();

    List<String> findApplicationsByServiceName(String serviceName);

    List<Consumer> findByApplication(String application);

    List<Consumer> findByAppandService(String app, String serviceName);

    List<String> findServicesByApplication(String application);

    List<String> findServices();

}
