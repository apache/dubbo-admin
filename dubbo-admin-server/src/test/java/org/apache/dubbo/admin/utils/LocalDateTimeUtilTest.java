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

import org.junit.Test;

import java.time.LocalDate;
import java.time.LocalDateTime;

import static org.hamcrest.CoreMatchers.is;
import static org.junit.Assert.assertThat;

public class LocalDateTimeUtilTest {
    @Test
    public void shouldGetLocalDateTime() {
        LocalDateTime localDateTime = LocalDateTimeUtil.formatToLDT("2020-01-02 10:11:12");

        assertThat(localDateTime.getYear(), is(2020));
        assertThat(localDateTime.getMonthValue(), is(1));
        assertThat(localDateTime.getDayOfMonth(), is(2));

        assertThat(localDateTime.getHour(), is(10));
        assertThat(localDateTime.getMinute(), is(11));
        assertThat(localDateTime.getSecond(), is(12));
    }

    @Test
    public void shouldGetLocalDate() {
        LocalDate localDate = LocalDateTimeUtil.formatToLD("2020-01-02");

        assertThat(localDate.getYear(), is(2020));
        assertThat(localDate.getMonthValue(), is(1));
        assertThat(localDate.getDayOfMonth(), is(2));
    }
}
