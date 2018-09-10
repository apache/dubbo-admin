//package org.apache.dubbo.admin.web.mvc.home;
//
//import org.apache.dubbo.admin.governance.service.ConsumerService;
//import org.apache.dubbo.admin.governance.service.ProviderService;
//import org.springframework.beans.factory.annotation.Autowired;
//import org.springframework.stereotype.Controller;
//import org.springframework.ui.Model;
//import org.springframework.web.bind.annotation.RequestMapping;
//import org.springframework.web.bind.annotation.RequestParam;
//
//import javax.servlet.http.HttpServletRequest;
//import javax.servlet.http.HttpServletResponse;
//
///**
// * @author zmx ON 2018/7/20
// */
//
//@Controller
//public class IndexController {
//
//    @Autowired
//    private ProviderService providerService;
//
//    @Autowired
//    private ConsumerService consumerService;
//
//    @RequestMapping("/")
//    public String search(@RequestParam(required = false) String filter,
//                                  @RequestParam(required = false, defaultValue = "") String pattern,
//                                  HttpServletRequest request,
//                                  HttpServletResponse response, Model model) {
//        if ("app".equals(pattern)) {
//            model.addAttribute("active", "app");
//        } else if ("ip".equals(pattern)) {
//            model.addAttribute("active", "ip");
//        } else {
//            model.addAttribute("active", "service");
//        }
//
//        return "serviceSearch";
//
//    }
//
//    @RequestMapping("/index")
//    public String index() {
//        return "index";
//    }
//}
