package org.apache.dubbo.admin.config;

import org.apache.dubbo.admin.common.exception.ConfigurationException;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.admin.registry.metadata.MetaDataCollector;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.registry.RegistryFactory;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.InjectMocks;
import org.powermock.api.mockito.PowerMockito;
import org.powermock.core.classloader.annotations.PowerMockIgnore;
import org.powermock.core.classloader.annotations.PrepareForTest;
import org.powermock.modules.junit4.PowerMockRunner;
import org.powermock.modules.junit4.PowerMockRunnerDelegate;
import org.springframework.test.context.junit4.SpringJUnit4ClassRunner;
import org.springframework.test.util.ReflectionTestUtils;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.fail;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.powermock.api.mockito.PowerMockito.mock;
import static org.powermock.api.mockito.PowerMockito.when;


@RunWith(PowerMockRunner.class)
@PowerMockRunnerDelegate(SpringJUnit4ClassRunner.class)
@PrepareForTest(value = {ExtensionLoader.class})
@PowerMockIgnore({"javax.management.*"})
public class ConfigCenterTest {

    @InjectMocks
    private ConfigCenter configCenter;

    @Test
    public void testGetDynamicConfiguration() throws Exception {
        // mock @value inject
        ReflectionTestUtils.setField(configCenter, "configCenter", "zookeeper://127.0.0.1:2181");
        ReflectionTestUtils.setField(configCenter, "group", "dubbo");
        ReflectionTestUtils.setField(configCenter, "username", "username");
        ReflectionTestUtils.setField(configCenter, "password", "password");

        ExtensionLoader extensionLoader = mock(ExtensionLoader.class);
        GovernanceConfiguration dynamicConfiguration = mock(GovernanceConfiguration.class);
        PowerMockito.mockStatic(ExtensionLoader.class);

        when(extensionLoader.getExtension(any())).thenReturn(dynamicConfiguration);
        when(ExtensionLoader.class, "getExtensionLoader", GovernanceConfiguration.class).thenReturn(extensionLoader);

        // stub config is null
        assertEquals(dynamicConfiguration, configCenter.getDynamicConfiguration());
        // stub config is registry address
        when(dynamicConfiguration.getConfig(anyString())).thenReturn("dubbo.registry.address=zookeeper://test-registry.com:2181");
        assertEquals(dynamicConfiguration, configCenter.getDynamicConfiguration());
        // stub config is meta date address
        when(dynamicConfiguration.getConfig(anyString())).thenReturn("dubbo.metadata-report.address=zookeeper://test-metadata.com:2181");
        assertEquals(dynamicConfiguration, configCenter.getDynamicConfiguration());

        // stub config is empty
        when(dynamicConfiguration.getConfig(anyString())).thenReturn("");
        assertEquals(dynamicConfiguration, configCenter.getDynamicConfiguration());

        // configCenter is null
        ReflectionTestUtils.setField(configCenter, "configCenter", null);
        // registryAddress is not null
        ReflectionTestUtils.setField(configCenter, "registryAddress", "zookeeper://127.0.0.1:2181");
        assertEquals(dynamicConfiguration, configCenter.getDynamicConfiguration());

        // configCenter, registryAddress are all null
        ReflectionTestUtils.setField(configCenter, "configCenter", null);
        ReflectionTestUtils.setField(configCenter, "registryAddress", null);
        try {
            configCenter.getDynamicConfiguration();
            fail("should throw exception when configCenter, registryAddress are all null");
        } catch (ConfigurationException e) {
        }
    }

    @Test
    public void testGetRegistry() throws Exception {
        try {
            configCenter.getRegistry();
            fail("should throw exception when registryAddress is blank");
        } catch (ConfigurationException e) {
        }

        // mock @value inject
        ReflectionTestUtils.setField(configCenter, "registryAddress", "zookeeper://127.0.0.1:2181");
        ReflectionTestUtils.setField(configCenter, "group", "dubbo");
        ReflectionTestUtils.setField(configCenter, "username", "username");
        ReflectionTestUtils.setField(configCenter, "password", "password");

        ExtensionLoader extensionLoader = mock(ExtensionLoader.class);
        RegistryFactory registryFactory = mock(RegistryFactory.class);
        Registry registry = mock(Registry.class);
        PowerMockito.mockStatic(ExtensionLoader.class);

        when(registryFactory.getRegistry(any(URL.class))).thenReturn(registry);
        when(extensionLoader.getAdaptiveExtension()).thenReturn(registryFactory);
        when(ExtensionLoader.class, "getExtensionLoader", RegistryFactory.class).thenReturn(extensionLoader);

        assertEquals(registry, configCenter.getRegistry());
    }

    @Test
    public void testGetMetadataCollector() throws Exception {
        // when metadataAddress is empty
        ReflectionTestUtils.setField(configCenter, "metadataAddress", "");
        configCenter.getMetadataCollector();

        // mock @value inject
        ReflectionTestUtils.setField(configCenter, "metadataAddress", "zookeeper://127.0.0.1:2181");
        ReflectionTestUtils.setField(configCenter, "group", "dubbo");
        ReflectionTestUtils.setField(configCenter, "username", "username");
        ReflectionTestUtils.setField(configCenter, "password", "password");

        ExtensionLoader extensionLoader = mock(ExtensionLoader.class);
        MetaDataCollector metaDataCollector = mock(MetaDataCollector.class);
        PowerMockito.mockStatic(ExtensionLoader.class);

        when(extensionLoader.getExtension(any())).thenReturn(metaDataCollector);
        when(ExtensionLoader.class, "getExtensionLoader", MetaDataCollector.class).thenReturn(extensionLoader);

        configCenter.getMetadataCollector();
    }
}
