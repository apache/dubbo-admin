package index

import (
	"github.com/apache/dubbo-admin/pkg/common/errors"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/duke-git/lancet/v2/slice"
	"k8s.io/client-go/tools/cache"
)

const ByMeshIndex = "idx_mesh"

func init() {
	rks := coremodel.ResourceSchemaRegistry().AllResourceKinds()
	slice.ForEach(rks, func(_ int, rk coremodel.ResourceKind) {
		RegisterIndexers(rk, map[string]cache.IndexFunc{
			ByMeshIndex: ByMesh,
		})
	})
}

func ByMesh(obj interface{}) ([]string, error) {
	r, ok := obj.(coremodel.Resource)
	if !ok {
		return nil, errors.NewAssertionError("Resource", obj)
	}
	return []string{r.MeshName()}, nil
}
