package org.apache.dubbo.admin.model.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * There is no description.
 *
 * @author XS <wanghaiqi@beeplay123.com>
 * @version 1.0
 * @date 2023/11/28 13:48
 */
@Data
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class AppConfigDTO {

    /**
     * Admin Title
     */
    private String adminTitle;
}
