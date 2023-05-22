package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"strings"
)

type RetryService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *RetryService) CreateOrUpdate(r *model.Retry) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(r.Service, r.Group, r.Version))
	newRule := r.ToRule()

	err := createOrUpdateOverride(key, "consumer", "retries", newRule)
	return err
}

func (tm *RetryService) Delete(r *model.Retry) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(r.Service, r.Group, r.Version))
	err2 := removeFromOverride(key, "consumer", "retries")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *RetryService) Search(r *model.Retry) ([]*model.Retry, error) {
	var result = make([]*model.Retry, 0)

	var con string
	if r.Service != "" {
		con = util.ColonSeparatedKey(r.Service, r.Group, r.Version)
	}

	list, err := getRules(con)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")
		retry := &model.Retry{
			Service: split[0],
			Group:   split[1],
			Version: split[2],
		}
		rv, err2 := getValue(v, "consumer", "retries")
		if err2 != nil {
			return result, err2
		}
		retry.Retry = rv.(int)
		result = append(result, retry)
	}

	return result, nil
}
