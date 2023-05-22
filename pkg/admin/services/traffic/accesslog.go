package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"strings"
)

type AccesslogService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *AccesslogService) CreateOrUpdate(a *model.Accesslog) error {
	key := services.GetOverridePath(a.Application)
	newRule := a.ToRule()

	err := createOrUpdateOverride(key, "provider", "accesslog", newRule)
	return err
}

func (tm *AccesslogService) Delete(a *model.Accesslog) error {
	key := services.GetOverridePath(a.Application)
	err2 := removeFromOverride(key, "provider", "accesslog")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *AccesslogService) Search(a *model.Accesslog) ([]*model.Accesslog, error) {
	var result = make([]*model.Accesslog, 0)

	list, err := getRules(a.Application)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		accesslog := &model.Accesslog{
			Application: a.Application,
		}
		alv, err2 := getValue(v, "provider", "accesslog")
		if err2 != nil {
			return result, err2
		}
		accesslog.Accesslog = alv.(string)
		result = append(result, accesslog)
	}

	return result, nil
}
