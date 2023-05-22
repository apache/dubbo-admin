package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"gopkg.in/yaml.v2"
	"strings"
)

type WeightService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *WeightService) CreateOrUpdate(p *model.Percentage) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(p.Service, p.Group, p.Version))
	newRule := p.ToRule()

	err := createOrUpdateOverride(key, "provider", "weight", newRule)
	return err
}

func (tm *WeightService) Delete(p *model.Percentage) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(p.Service, p.Group, p.Version))
	err := removeFromOverride(key, "provider", "weight")
	if err != nil {
		return err
	}
	return nil
}

func (tm *WeightService) Search(p *model.Percentage) ([]*model.Percentage, error) {
	var result = make([]*model.Percentage, 0)

	var con string
	if p.Service != "" {
		con = util.ColonSeparatedKey(p.Service, p.Group, p.Version)
	}

	list, err := getRules(con)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")
		percentage := &model.Percentage{
			Service: split[0],
			Group:   split[1],
			Version: split[2],
			Weights: make([]model.Weight, 0),
		}

		override := &model.Override{}
		err = yaml.Unmarshal([]byte(v), override)
		if err != nil {
			return result, err
		}
		for _, c := range override.Configs {
			if c.Side == "provider" && c.Parameters["weight"] != "" {
				percentage.Weights = append(percentage.Weights, model.Weight{
					Weight: c.Parameters["weight"].(int),
					Match:  c.Match,
				})
			}
		}

		result = append(result, percentage)
	}

	return result, nil
}
