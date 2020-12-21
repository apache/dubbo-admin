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
package org.apache.dubbo.admin.utils;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;


/**
 * Date time tool class of LocalDateTime.
 */
public class LocalDateTimeUtil {

    /**
     * Create a LocalDateTime based on the passed in string date and the corresponding format.
     * 2020/11/2 10:58
     *
     * @param dateStr
     * @param dateFormatter
     * @return java.time.LocalDateTime
     */
    public static LocalDateTime formatToLDT(String dateStr, String dateFormatter) {
        //yyyy-MM-dd HH:mm:ss
        return LocalDateTime.parse(dateStr, DateTimeFormatter.ofPattern(dateFormatter));
    }

    /**
     * Create LocalDateTime from a string in the format "yyyy-MM-dd HH:mm:ss".
     * 2020/11/2 10:58
     *
     * @param dateStr
     * @return java.time.LocalDateTime
     */
    public static LocalDateTime formatToLDT(String dateStr) {
        return formatToLDT(dateStr, "yyyy-MM-dd HH:mm:ss");
    }

    /**
     * Create a LocalDate based on the passed in string date and the corresponding format.
     * 2020/11/2 10:58
     *
     * @param dateStr
     * @param dateFormatter
     * @return java.time.LocalDate
     */
    public static LocalDate formatToLD(String dateStr, String dateFormatter) {
        return LocalDate.parse(dateStr, DateTimeFormatter.ofPattern(dateFormatter));
    }

    /**
     * Create LocalDate from a string in the format "yyyy-MM-dd".
     * 2020/11/2 10:58
     *
     * @param dateStr
     * @return java.time.LocalDate
     */
    public static LocalDate formatToLD(String dateStr) {
        return formatToLD(dateStr, "yyyy-MM-dd");
    }


}
