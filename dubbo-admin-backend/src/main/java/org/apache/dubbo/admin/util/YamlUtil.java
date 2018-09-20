package org.apache.dubbo.admin.util;

import org.yaml.snakeyaml.Yaml;

import java.util.Map;

public class YamlUtil {

    private static Yaml yaml;

    static {
        yaml = new Yaml();
    }

    public static Map<String, Object> loadString(String text) {
        if (text != null) {
            return yaml.load(text);
        }
        return null;
    }

}
