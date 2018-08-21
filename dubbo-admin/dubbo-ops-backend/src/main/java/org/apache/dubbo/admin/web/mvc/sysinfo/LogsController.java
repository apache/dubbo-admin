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
package org.apache.dubbo.admin.web.mvc.sysinfo;

import com.alibaba.dubbo.common.logger.Level;
import com.alibaba.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.admin.web.mvc.BaseController;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.RequestMapping;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.File;
import java.io.FileInputStream;
import java.nio.ByteBuffer;
import java.nio.channels.FileChannel;
import java.text.SimpleDateFormat;
import java.util.Date;

@Controller
@RequestMapping("/sysinfo/logs")
public class LogsController extends BaseController {

    private static final int SHOW_LOG_LENGTH = 30000;

    @RequestMapping("")
    public String index(HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        prepare(request, response, model, "index", "logs");
        long size;
        String content;
        String modified;
        File file = LoggerFactory.getFile();
        if (file != null && file.exists()) {
            FileInputStream fis = new FileInputStream(file);
            FileChannel channel = fis.getChannel();
            size = channel.size();
            ByteBuffer bb;
            if (size <= SHOW_LOG_LENGTH) {
                bb = ByteBuffer.allocate((int) size);
                channel.read(bb, 0);
            } else {
                int pos = (int) (size - SHOW_LOG_LENGTH);
                bb = ByteBuffer.allocate(SHOW_LOG_LENGTH);
                channel.read(bb, pos);
            }
            bb.flip();
            content = new String(bb.array()).replace("<", "&lt;").replace(">", "&gt;");
            modified = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss").format(new Date(file.lastModified()));
        } else {
            size = 0;
            content = "";
            modified = "Not exist";
        }
        Level level = LoggerFactory.getLevel();
        model.addAttribute("name", file == null ? "" : file.getAbsoluteFile());
        model.addAttribute("size", String.valueOf(size));
        model.addAttribute("level", level == null ? "" : level);
        model.addAttribute("modified", modified);
        model.addAttribute("content", content);
        return "sysinfo/screen/logs/index";
    }
}
