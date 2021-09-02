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

import org.apache.dubbo.admin.annotation.Authority;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.common.exception.VersionValidationException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.model.dto.MeshRouteDTO;
import org.apache.dubbo.admin.service.MeshRouteService;
import org.apache.dubbo.admin.service.ProviderService;

import org.apache.commons.lang3.StringUtils;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import java.util.ArrayList;
import java.util.List;

@Authority(needLogin = true)
@RestController
@RequestMapping("/api/{env}/rules/route/mesh")
public class MeshRouteController {

    private final MeshRouteService meshRouteService;

    private final ProviderService providerService;

    public MeshRouteController(MeshRouteService meshRouteService, ProviderService providerService) {
        this.meshRouteService = meshRouteService;
        this.providerService = providerService;
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createMeshRoute(@RequestBody MeshRouteDTO meshRoute, @PathVariable String env) {
        String app = meshRoute.getApplication();
        if (StringUtils.isEmpty(app)) {
            throw new ParamValidationException("app is Empty!");
        }
        if (providerService.findVersionInApplication(app).startsWith("2")) {
            throw new VersionValidationException("dubbo 2.x does not support mesh route");
        }

        return meshRouteService.createMeshRule(meshRoute);
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateRule(@PathVariable String id, @RequestBody MeshRouteDTO meshRoute, @PathVariable String env) {
        id = id.replace(Constants.ANY_VALUE, Constants.PATH_SEPARATOR);
        if (meshRouteService.findMeshRoute(id) == null) {
            throw new ResourceNotFoundException("can not find mesh route, Id: " + id);
        }
        meshRoute.setId(id);
        return meshRouteService.updateMeshRule(meshRoute);
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<MeshRouteDTO> searchRoutes(@RequestParam String application, @PathVariable String env) {
        if (StringUtils.isBlank(application)) {
            throw new ParamValidationException("application is required.");
        }
        List<MeshRouteDTO> result = new ArrayList<>();

        MeshRouteDTO meshRoute = meshRouteService.findMeshRoute(application);
        if (meshRoute != null) {
            result.add(meshRoute);
        }
        return result;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public MeshRouteDTO detailRoute(@PathVariable String id, @PathVariable String env) {
        id = id.replace(Constants.ANY_VALUE, Constants.PATH_SEPARATOR);
        MeshRouteDTO meshRoute = meshRouteService.findMeshRoute(id);
        if (meshRoute == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        return meshRoute;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteRoute(@PathVariable String id, @PathVariable String env) {
        id = id.replace(Constants.ANY_VALUE, Constants.PATH_SEPARATOR);
        return meshRouteService.deleteMeshRule(id);
    }

}
