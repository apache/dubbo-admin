package manager

import (
	"github.com/apache/dubbo-admin/pkg/common/errors"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// GetByKey is a helper function of ResourceManager.GeyByKey
func GetByKey[T model.Resource](rm ReadOnlyResourceManager, rk model.ResourceKind, key string) (r T, exist bool, err error) {
	resource, exist, err := rm.GetByKey(rk, key)
	if err != nil || !exist {
		var zero T
		return zero, exist, err
	}

	typedResource, ok := resource.(T)
	if !ok {
		var zero T
		return zero, false, errors.NewAssertionError(rk, typedResource.ResourceKind())
	}
	return typedResource, true, nil
}

// ListByIndexes is a helper function of ResourceManager.ListByIndexes
func ListByIndexes[T model.Resource](rm ReadOnlyResourceManager, rk model.ResourceKind, indexes map[string]interface{}) ([]T, error) {
	resources, err := rm.ListByIndexes(rk, indexes)
	if err != nil {
		return nil, err
	}

	typedResources := make([]T, len(resources))
	for i, resource := range resources {
		typedResource, ok := resource.(T)
		if !ok {
			return nil, errors.NewAssertionError(rk, typedResource.ResourceKind())
		}
		typedResources[i] = typedResource
	}

	return typedResources, nil
}

// PageListByIndexes is a helper function of ResourceManager.PageListByIndexes
func PageListByIndexes[T model.Resource](
	rm ReadOnlyResourceManager,
	rk model.ResourceKind,
	indexes map[string]interface{},
	pr model.PageReq) (*model.PageData[T], error) {

	pageData, err := rm.PageListByIndexes(rk, indexes, pr)
	if err != nil {
		return nil, err
	}

	typedResources := make([]T, len(pageData.Data))
	for i, resource := range pageData.Data {
		typedResource, ok := resource.(T)
		if !ok {
			return nil, errors.NewAssertionError(rk, typedResource.ResourceKind())
		}
		typedResources[i] = typedResource
	}
	newPageData := &model.PageData[T]{
		Pagination: model.Pagination{
			Total:      pageData.Total,
			PageOffset: pageData.PageOffset,
			PageSize:   pageData.PageSize,
		},
		Data: typedResources,
	}
	return newPageData, nil
}

// PageSearchResourceByConditions is a helper function of ResourceManager.PageSearchResourceByConditions
func PageSearchResourceByConditions[T model.Resource](
	rm ReadOnlyResourceManager,
	rk model.ResourceKind,
	conditions []string,
	pr model.PageReq) (*model.PageData[T], error) {
	pageData, err := rm.PageSearchResourceByConditions(rk, conditions, pr)
	if err != nil {
		return nil, err
	}

	typedResources := make([]T, len(pageData.Data))
	for i, resource := range pageData.Data {
		typedResource, ok := resource.(T)
		if !ok {
			return nil, errors.NewAssertionError(rk, typedResource.ResourceKind())
		}
		typedResources[i] = typedResource
	}
	newPageData := &model.PageData[T]{
		Pagination: model.Pagination{
			Total:      pageData.Total,
			PageOffset: pageData.PageOffset,
			PageSize:   pageData.PageSize,
		},
		Data: typedResources,
	}
	return newPageData, nil
}
