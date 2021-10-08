CREATE TABLE `mock_rule` (
    `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '主键',
    `service_name` varchar(255) DEFAULT NULL COMMENT '服务名',
    `method_name` varchar(255) DEFAULT NULL COMMENT '方法名',
    `rule` text COMMENT '规则',
    `enable` tinyint(1) NOT NULL DEFAULT '1',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`)
);