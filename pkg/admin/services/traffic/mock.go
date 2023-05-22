package traffic

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"strings"
)

type MockService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *MockService) CreateOrUpdate(m *model.Mock) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(m.Service, m.Group, m.Version))
	newRule := m.ToRule()

	err := createOrUpdateOverride(key, "consumer", "mock", newRule)
	return err
}

func (tm *MockService) Delete(m *model.Mock) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(m.Service, m.Group, m.Version))
	err2 := removeFromOverride(key, "consumer", "mock")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *MockService) Search(m *model.Mock) ([]*model.Mock, error) {
	var result = make([]*model.Mock, 0)

	var con string
	if m.Service != "" {
		con = util.ColonSeparatedKey(m.Service, m.Group, m.Version)
	}
	list, err := getRules(con)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")
		mock := &model.Mock{
			Service: split[0],
			Group:   split[1],
			Version: split[2],
		}
		mv, err2 := getValue(v, "consumer", "mock")
		if err2 != nil {
			return result, err2
		}
		mock.Mock = mv.(string)
		result = append(result, mock)
	}

	return result, nil
}
