package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"gopkg.in/yaml.v2"
	"strings"
)

type ArgumentService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *ArgumentService) CreateOrUpdate(a *model.Argument) error {
	key := services.GetRoutePath(util.ColonSeparatedKey(a.Service, a.Group, a.Version), constant.ConditionRoute)
	newRule := a.ToRule()

	err := createOrUpdateCondition(key, newRule)
	return err
}

func (tm *ArgumentService) Delete(a *model.Argument) error {
	key := services.GetRoutePath(util.ColonSeparatedKey(a.Service, a.Group, a.Version), constant.ConditionRoute)
	err2 := removeCondition(key, a.Rule)
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *ArgumentService) Search(a *model.Argument) ([]*model.Argument, error) {
	var result = make([]*model.Argument, 0)

	var con string
	if a.Service != "" {
		con = util.ColonSeparatedKey(a.Service, a.Group, a.Version)
	}

	list, err := getRules(con)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConditionRuleSuffix)
		split := strings.Split(k, ":")
		argument := &model.Argument{
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
				argument.Rule = c
				break
			}
		}

		result = append(result, argument)
	}

	return result, nil
}
