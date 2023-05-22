package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"gopkg.in/yaml.v2"
	"strings"
)

type GrayService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *GrayService) CreateOrUpdate(g *model.Gray) error {
	key := services.GetRoutePath(g.Application, constant.TagRuleSuffix)
	newRule := g.ToRule()

	err := createOrUpdateTag(key, newRule)
	return err
}

func (tm *GrayService) Delete(g *model.Gray) error {
	key := services.GetRoutePath(g.Application, constant.TagRuleSuffix)
	err2 := deleteTag(key)
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *GrayService) Search(g *model.Gray) ([]*model.Gray, error) {
	var result = make([]*model.Gray, 0)

	list, err := getRules(g.Application)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.TagRuleSuffix)
		gray := &model.Gray{
			Application: g.Application,
		}

		route := &model.TagRoute{}
		err = yaml.Unmarshal([]byte(v), route)
		if err != nil {
			return result, err
		}
		gray.Tags = route.Tags
		result = append(result, gray)
	}

	return result, nil
}
