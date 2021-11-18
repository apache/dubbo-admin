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

package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.AbstractSpringIntegrationTest;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.model.dto.MeshRouteDTO;
import org.apache.dubbo.admin.service.ProviderService;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.After;
import org.junit.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertTrue;
import static org.mockito.Mockito.when;


public class MeshRouteControllerTest extends AbstractSpringIntegrationTest {

    private final String env = "whatever";

    @Autowired
    private ObjectMapper objectMapper;

    @MockBean
    private ProviderService providerService;

    @After
    public void tearDown() throws Exception {
        if (zkClient.checkExists().forPath("/dubbo") != null) {
            zkClient.delete().deletingChildrenIfNeeded().forPath("/dubbo");
        }
    }

    private String getFileContent(String file) throws IOException {
        try (InputStream stream = this.getClass().getResourceAsStream(file)) {
            byte[] bytes = new byte[stream.available()];
            stream.read(bytes);
            return new String(bytes, StandardCharsets.UTF_8);
        }
    }


    @Test
    public void createMeshRoute() throws IOException {
        MeshRouteDTO meshRoute = new MeshRouteDTO();
        ResponseEntity<String> response;
        String application = "mesh-create";
        // application are all blank
        response = restTemplate.postForEntity(url("/api/{env}/rules/route/mesh"), meshRoute, String.class, env);
        assertFalse("should return a fail response, when application is blank", (Boolean) objectMapper.readValue(response.getBody(), Map.class).get("success"));

        // valid mesh rule
        meshRoute.setApplication(application);
        meshRoute.setMeshRule(getFileContent("/MeshRoute.yml"));
        when(providerService.findVersionInApplication(application)).thenReturn("3.0.0");
        response = restTemplate.postForEntity(url("/api/{env}/rules/route/mesh"), meshRoute, String.class, env);
        assertEquals(HttpStatus.CREATED, response.getStatusCode());
        assertTrue(Boolean.valueOf(response.getBody()));
    }


    @Test
    public void detailMeshRoute() throws Exception {
        String id = "1";
        ResponseEntity<String> response;
        // when balancing is not exist
        response = restTemplate.getForEntity(url("/api/{env}/rules/route/mesh/{id}"), String.class, env, id);
        assertFalse("should return a fail response, when id is null", (Boolean) objectMapper.readValue(response.getBody(), Map.class).get("success"));
        // when balancing is not null
        String application = "mesh-detail";
        String content = getFileContent("/MeshRoute.yml");
        String path = "/dubbo/" + Constants.CONFIG_KEY + Constants.PATH_SEPARATOR + application + Constants.MESH_RULE_SUFFIX;
        zkClient.create().creatingParentContainersIfNeeded().forPath(path);
        zkClient.setData().forPath(path, content.getBytes());
        assertNotNull("zk path should not be null before deleting", zkClient.checkExists().forPath(path));

        response = restTemplate.getForEntity(url("/api/{env}/rules/route/mesh/{id}"), String.class, env, application);
        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertEquals(content, objectMapper.readValue(response.getBody(), Map.class).get("meshRule"));
    }

    @Test
    public void updateMeshRoute() throws Exception {
        String application = "mesh-update";
        String content = getFileContent("/MeshRoute.yml");
        String path = "/dubbo/" + Constants.CONFIG_KEY + Constants.PATH_SEPARATOR + application + Constants.MESH_RULE_SUFFIX;
        zkClient.create().creatingParentContainersIfNeeded().forPath(path);
        zkClient.setData().forPath(path, content.getBytes());
        assertNotNull("zk path should not be null before deleting", zkClient.checkExists().forPath(path));

        MeshRouteDTO meshRoute = new MeshRouteDTO();
        meshRoute.setApplication(application);
        meshRoute.setMeshRule(getFileContent("/MeshRouteTest2.yml"));

        ResponseEntity<String> response = restTemplate.exchange(url("/api/{env}/rules/route/mesh/{id}"), HttpMethod.PUT, new HttpEntity<>(meshRoute, null), String.class, env, application);
        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertTrue(Boolean.valueOf(response.getBody()));
        byte[] bytes = zkClient.getData().forPath(path);
        String updatedConfig = new String(bytes);
        assertEquals(updatedConfig, meshRoute.getMeshRule());
    }


    @Test
    public void deleteMeshRoute() throws Exception {
        String application = "mesh-delete";
        String content = getFileContent("/MeshRoute.yml");
        String path = "/dubbo/" + Constants.CONFIG_KEY + Constants.PATH_SEPARATOR + application + Constants.MESH_RULE_SUFFIX;
        zkClient.create().creatingParentContainersIfNeeded().forPath(path);
        zkClient.setData().forPath(path, content.getBytes());
        assertNotNull("zk path should not be null before deleting", zkClient.checkExists().forPath(path));

        ResponseEntity<String> response = restTemplate.exchange(url("/api/{env}/rules/route/mesh/{id}"), HttpMethod.DELETE, new HttpEntity<>(null), String.class, env, application);
        assertEquals(HttpStatus.OK, response.getStatusCode());
        assertNull(zkClient.checkExists().forPath(path));
    }
}
