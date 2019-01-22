package org.apache.dubbo.admin.common.util;

import org.junit.Test;

import static org.junit.Assert.*;

public class CoderUtilTest {

    @Test
    public void MD5_16bit() {
        assertNull(CoderUtil.MD5_16bit(null));

        String input = "dubbo";
        String output = "2CC9DEED96FE012E";
        assertEquals(output, CoderUtil.MD5_16bit(input));
    }

    @Test
    public void MD5_32bit() {
        String input = null;
        assertNull(CoderUtil.MD5_32bit(input));

        input = "dubbo";
        String output = "AA4E1B8C2CC9DEED96FE012EF2E0752A";
        assertEquals(output, CoderUtil.MD5_32bit(input));
    }

    @Test
    public void MD5_32bit1() {
        byte[] input = null;
        assertNull(CoderUtil.MD5_32bit(input));

        input = "dubbo".getBytes();
        String output = "AA4E1B8C2CC9DEED96FE012EF2E0752A";
        assertEquals(output, CoderUtil.MD5_32bit(input));
    }

    @Test
    public void decodeBase64() {
        try {
            CoderUtil.decodeBase64(null);
            fail("when param is null, this should throw exception");
        } catch (Exception e) {
        }

        String input = "ZHViYm8=";
        String output = "dubbo";
        assertEquals(output, CoderUtil.decodeBase64(input));
    }
}