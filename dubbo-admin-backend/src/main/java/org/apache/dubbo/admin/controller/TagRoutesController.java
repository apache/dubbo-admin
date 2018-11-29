package org.apache.dubbo.admin.controller;

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.model.domain.TagRoute;
import org.apache.dubbo.admin.model.dto.RouteDTO;
import org.apache.dubbo.admin.model.dto.TagRouteDTO;
import org.apache.dubbo.admin.service.RouteService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/{env}/rules/route/tag")
public class TagRoutesController {


    private final RouteService routeService;

    @Autowired
    public TagRoutesController(RouteService routeService) {
        this.routeService = routeService;
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createRule(@RequestBody TagRouteDTO routeDTO, @PathVariable String env) {
        String app = routeDTO.getApplication();
        if (StringUtils.isEmpty(app)) {
            throw new ParamValidationException("app is Empty!");
        }
        TagRoute tagRoute = convertTagRouteDTOToTagRoute(routeDTO);
        routeService.createTagRoute(tagRoute);
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateRule(@PathVariable String id, @RequestBody TagRouteDTO routeDTO, @PathVariable String dev) {
        if (routeService.findConditionRoute(id) == null) {
            //throw exception
        }
        TagRoute tagRoute = convertTagRouteDTOToTagRoute(routeDTO);
        routeService.updateTagRoute(tagRoute);
        return true;

    }

    @RequestMapping(method = RequestMethod.GET)
    public TagRouteDTO searchRoutes(@RequestParam String application, @PathVariable String env) {
        TagRoute tagRoute = null;
        if (StringUtils.isNotEmpty(application)) {
            tagRoute = routeService.findTagRoute(application);
        }
        if (tagRoute != null) {
            TagRouteDTO routeDTO = convertTagRouteToTagRouteDTO(tagRoute);
            return routeDTO;
        }
        return null;

    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public TagRouteDTO detailRoute(@PathVariable String id, @PathVariable String env) {
        TagRoute tagRoute = routeService.findTagRoute(id);
        if (tagRoute == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        TagRouteDTO tagRouteDTO = convertTagRouteToTagRouteDTO(tagRoute);
        return tagRouteDTO;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteRoute(@PathVariable String id, @PathVariable String env) {
        routeService.deleteTagRoute(id);
        return true;
    }

    @RequestMapping(value = "/enable/{id}", method = RequestMethod.PUT)
    public boolean enableRoute(@PathVariable String id, @PathVariable String env) {
        routeService.enableTagRoute(id);
        return true;
    }

    @RequestMapping(value = "/disable/{id}", method = RequestMethod.PUT)
    public boolean disableRoute(@PathVariable String id, @PathVariable String env) {
        routeService.disableTagRoute(id);
        return true;
    }

    private TagRouteDTO convertTagRouteToTagRouteDTO(TagRoute tagRoute) {
        TagRouteDTO tagRouteDTO = new TagRouteDTO();
        tagRouteDTO.setTags(tagRoute.getTags());
        tagRouteDTO.setApplication(tagRoute.getKey());
        tagRouteDTO.setEnabled(tagRoute.isEnabled());
        tagRouteDTO.setForce(tagRoute.isForce());
        tagRouteDTO.setPriority(tagRoute.getPriority());
        tagRouteDTO.setRuntime(tagRoute.isRuntime());
        return tagRouteDTO;
    }

    private TagRoute convertTagRouteDTOToTagRoute(TagRouteDTO tagRouteDTO) {
        TagRoute tagRoute = new TagRoute();
        tagRoute.setEnabled(tagRouteDTO.isEnabled());
        tagRoute.setForce(tagRouteDTO.isForce());
        tagRoute.setKey(tagRouteDTO.getApplication());
        tagRoute.setPriority(tagRouteDTO.getPriority());
        tagRoute.setRuntime(tagRouteDTO.isRuntime());
        tagRoute.setTags(tagRouteDTO.getTags());
        return tagRoute;
    }

}

