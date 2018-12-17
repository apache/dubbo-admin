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

package org.apache.dubbo.admin.common.util;

import org.apache.dubbo.admin.model.dto.DynamicConfigDTO;
import org.junit.Test;

import java.io.IOException;
import java.io.InputStream;

public class YamlParserTest {

    private String streamToString(InputStream stream) {
        try {
            byte[] bytes = new byte[stream.available()];
            stream.read(bytes);
            return new String(bytes);
        } catch (IOException e) {
            e.printStackTrace();
        }
        return null;
    }

    @Test
    public void parseLoadBalanceTest() {
        InputStream yamlStream = this.getClass().getResourceAsStream("/LoadBalance.yml");
        DynamicConfigDTO overrideDTO = YamlParser.loadObject(streamToString(yamlStream), DynamicConfigDTO.class);
        if ("weight".equals(null)) {
            System.out.println("aa");
        }
        overrideDTO.getConfigs();
    }


}
