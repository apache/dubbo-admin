package domain

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	cons "github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"net/url"

	"strconv"
	"strings"
)

type Override struct {
	Entity
	Service     string
	Address     string
	Enabled     bool
	Application string
	Params      string
}

func (o Override) GetService() string {
	return o.Service
}

func (o Override) ToURL() *common.URL {
	group := util.GetGroup(o.GetService())
	version := util.GetVersion(o.Service)
	interfaceName := util.GetInterface(o.Service)
	var sb strings.Builder
	sb.WriteString(constant.OverrideProtocol)
	sb.WriteString("://")
	if o.Address != "" && o.Address != constant.AnyValue {
		sb.WriteString(o.Address)
	} else {
		sb.WriteString(constant.AnyhostKey)
	}
	sb.WriteString("/")
	sb.WriteString(interfaceName)
	sb.WriteString("?")
	params, _ := url.ParseQuery(o.Params)
	params.Set(constant.CategoryKey, constant.ConfiguratorsCategory)
	params.Set(constant.EnabledKey, strconv.FormatBool(o.Enabled))
	params.Set(cons.DynamicKey, "false")
	if o.Application != "" && o.Application != constant.AnyValue {
		params.Set(constant.ApplicationKey, o.Application)
	}
	if group != "" {
		params.Set(constant.GroupKey, group)
	}
	if version != "" {
		params.Set(constant.VersionKey, version)
	}
	sb.WriteString(params.Encode())

	// TODO 字符串转换成common.URL
	_, _ = url.Parse(sb.String())
	//var urlNew common.URL
	//urlNew = urltemp
	//return
	return nil
}
