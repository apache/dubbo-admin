package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"gopkg.in/yaml.v2"
	"strings"
)

type RegionService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *RegionService) CreateOrUpdate(r *model.Region) error {
	key := services.GetRoutePath(util.ColonSeparatedKey(r.Service, r.Group, r.Version), constant.ConditionRoute)
	newRule := r.ToRule()

	err := createOrUpdateCondition(key, newRule)
	return err
}

func (tm *RegionService) Delete(r *model.Region) error {
	key := services.GetRoutePath(util.ColonSeparatedKey(r.Service, r.Group, r.Version), constant.ConditionRoute)
	err2 := removeCondition(key, r.Rule)
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *RegionService) Search(r *model.Region) ([]*model.Region, error) {
	var result = make([]*model.Region, 0)

	var con string
	if r.Service != "" {
		con = util.ColonSeparatedKey(r.Service, r.Group, r.Version)
	}

	list, err := getRules(con)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConditionRuleSuffix)
		split := strings.Split(k, ":")
		region := &model.Region{
			Service: split[0],
			Group:   split[1],
			Version: split[2],
		}

		route := &model.ConditionRoute{}
		err = yaml.Unmarshal([]byte(v), route)
		if err != nil {
			return result, err
		}
		for _, c := range route.Conditions {
			//fixme, regex match
			if strings.Contains(c, model.AdminIdentifier) {
				region.Rule = c
				break
			}
		}

		result = append(result, region)
	}

	return result, nil
}
