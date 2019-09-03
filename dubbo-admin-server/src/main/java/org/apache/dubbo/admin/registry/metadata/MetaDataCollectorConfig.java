package org.apache.dubbo.admin.registry.metadata;

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.configcenter.DynamicConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;

public class MetaDataCollectorConfig extends org.apache.dubbo.config.MetadataReportConfig {
	
	private static final long serialVersionUID = -8840658169638705990L;

	@Bean
	@ConditionalOnMissingBean(DynamicConfiguration.class)
	public MetaDataCollector metaDataCollector() {
		URL metadataUrl = formUrl();
		MetaDataCollector metaDataCollector = ExtensionLoader.getExtensionLoader(MetaDataCollector.class)
				.getExtension(metadataUrl.getProtocol());
		metaDataCollector.setUrl(metadataUrl);
		metaDataCollector.init();
		return metaDataCollector;
	}

	@Bean
	@ConditionalOnMissingBean
	public MetaDataCollector configCenterMetaDataCollector(DynamicConfiguration config) {
		super.refresh();
		return metaDataCollector();
	}

	private URL formUrl() {
		URL url = URL.valueOf(getAddress());
		if (StringUtils.isNotEmpty(getGroup())) {
			url = url.addParameter(Constants.GROUP_KEY, getGroup());
		}
		if (StringUtils.isNotEmpty(getUsername())) {
			url = url.setUsername(getUsername());
		}
		if (StringUtils.isNotEmpty(getPassword())) {
			url = url.setPassword(getPassword());
		}
		url = url.addParameter("namespace", getParameters().get("namespace"));
		return url;
	}
}
