package reports

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	mesh_proto "github.com/kumahq/kuma/api/mesh/v1alpha1"
	kuma_cp "github.com/kumahq/kuma/pkg/config/app/kuma-cp"
	config_core "github.com/kumahq/kuma/pkg/config/core"
	"github.com/kumahq/kuma/pkg/core"
	"github.com/kumahq/kuma/pkg/core/resources/apis/mesh"
	"github.com/kumahq/kuma/pkg/core/resources/apis/system"
	"github.com/kumahq/kuma/pkg/core/resources/registry"
	core_runtime "github.com/kumahq/kuma/pkg/core/runtime"
	"github.com/kumahq/kuma/pkg/core/user"
	kuma_version "github.com/kumahq/kuma/pkg/version"
)

const (
	pingInterval = 3600
	pingHost     = "kong-hf.konghq.com"
	pingPort     = 61832
)

var (
	log = core.Log.WithName("core").WithName("reports")
)

/*
  - buffer initialized upon Init call
  - append adds more keys onto it
*/

type reportsBuffer struct {
	sync.Mutex
	mutable   map[string]string
	immutable map[string]string
}

func fetchDataplanes(ctx context.Context, rt core_runtime.Runtime) (*mesh.DataplaneResourceList, error) {
	dataplanes := mesh.DataplaneResourceList{}
	if err := rt.ReadOnlyResourceManager().List(ctx, &dataplanes); err != nil {
		return nil, errors.Wrap(err, "could not fetch dataplanes")
	}

	return &dataplanes, nil
}

func fetchMeshes(ctx context.Context, rt core_runtime.Runtime) (*mesh.MeshResourceList, error) {
	meshes := mesh.MeshResourceList{}
	if err := rt.ReadOnlyResourceManager().List(ctx, &meshes); err != nil {
		return nil, errors.Wrap(err, "could not fetch meshes")
	}

	return &meshes, nil
}

func fetchZones(ctx context.Context, rt core_runtime.Runtime) (*system.ZoneResourceList, error) {
	zones := system.ZoneResourceList{}
	if err := rt.ReadOnlyResourceManager().List(ctx, &zones); err != nil {
		return nil, errors.Wrap(err, "could not fetch zones")
	}
	return &zones, nil
}

func fetchNumPolicies(ctx context.Context, rt core_runtime.Runtime) (map[string]string, error) {
	policyCounts := map[string]string{}

	for _, descr := range registry.Global().ObjectDescriptors() {
		typedList := descr.NewList()
		k := "n_" + strings.ToLower(string(descr.Name))
		if err := rt.ReadOnlyResourceManager().List(ctx, typedList); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not fetch %s", k))
		}
		policyCounts[k] = strconv.Itoa(len(typedList.GetItems()))
	}
	return policyCounts, nil
}

func fetchNumOfServices(ctx context.Context, rt core_runtime.Runtime) (int, int, error) {
	insights := mesh.ServiceInsightResourceList{}
	if err := rt.ReadOnlyResourceManager().List(ctx, &insights); err != nil {
		return 0, 0, errors.Wrap(err, "could not fetch service insights")
	}
	internalServices := 0
	for _, insight := range insights.Items {
		internalServices += len(insight.Spec.Services)
	}

	externalServicesList := mesh.ExternalServiceResourceList{}
	if err := rt.ReadOnlyResourceManager().List(ctx, &externalServicesList); err != nil {
		return 0, 0, errors.Wrap(err, "could not fetch external services")
	}
	return internalServices, len(externalServicesList.Items), nil
}

func (b *reportsBuffer) marshall() (string, error) {
	var builder strings.Builder

	_, err := fmt.Fprintf(&builder, "<14>")
	if err != nil {
		return "", err
	}

	for k, v := range b.immutable {
		_, err := fmt.Fprintf(&builder, "%s=%s;", k, v)
		if err != nil {
			return "", err
		}
	}

	for k, v := range b.mutable {
		_, err := fmt.Fprintf(&builder, "%s=%s;", k, v)
		if err != nil {
			return "", err
		}
	}

	return builder.String(), nil
}

// XXX this function retrieves all dataplanes and all meshes;
// ideally, the number of dataplanes and number of meshes
// should be pushed from the outside rather than pulled
func (b *reportsBuffer) updateEntitiesReport(rt core_runtime.Runtime) error {
	ctx := user.Ctx(context.Background(), user.ControlPlane)
	dps, err := fetchDataplanes(ctx, rt)
	if err != nil {
		return err
	}
	b.mutable["dps_total"] = strconv.Itoa(len(dps.Items))

	ngateways := 0
	gatewayTypes := map[string]int{}
	for _, dp := range dps.Items {
		spec := dp.GetSpec().(*mesh_proto.Dataplane)
		gateway := spec.GetNetworking().GetGateway()
		if gateway != nil {
			ngateways++
			gatewayType := strings.ToLower(gateway.GetType().String())
			gatewayTypes["gateway_dp_type_"+gatewayType] += 1
		}
	}
	b.mutable["gateway_dps"] = strconv.Itoa(ngateways)
	for gtype, n := range gatewayTypes {
		b.mutable[gtype] = strconv.Itoa(n)
	}

	meshes, err := fetchMeshes(ctx, rt)
	if err != nil {
		return err
	}
	b.mutable["meshes_total"] = strconv.Itoa(len(meshes.Items))

	switch rt.Config().Mode {
	case config_core.Standalone:
		b.mutable["zones_total"] = strconv.Itoa(1)
	case config_core.Global:
		zones, err := fetchZones(ctx, rt)
		if err != nil {
			return err
		}
		b.mutable["zones_total"] = strconv.Itoa(len(zones.Items))
	}

	internalServices, externalServices, err := fetchNumOfServices(ctx, rt)
	if err != nil {
		return err
	}
	b.mutable["internal_services"] = strconv.Itoa(internalServices)
	b.mutable["external_services"] = strconv.Itoa(externalServices)
	b.mutable["services_total"] = strconv.Itoa(internalServices + externalServices)

	policyCounts, err := fetchNumPolicies(ctx, rt)
	if err != nil {
		return err
	}

	for k, count := range policyCounts {
		b.mutable[k] = count
	}
	return nil
}

func (b *reportsBuffer) dispatch(rt core_runtime.Runtime, host string, port int, pingType string, extraFn core_runtime.ExtraReportsFn) error {
	if err := b.updateEntitiesReport(rt); err != nil {
		return err
	}
	b.mutable["signal"] = pingType
	b.mutable["cluster_id"] = rt.GetClusterId()
	b.mutable["uptime"] = strconv.FormatInt(int64(time.Since(rt.GetStartTime())/time.Second), 10)
	if extraFn != nil {
		if valMap, err := extraFn(rt); err != nil {
			return err
		} else {
			b.Append(valMap)
		}
	}
	pingData, err := b.marshall()
	if err != nil {
		return err
	}

	conf := &tls.Config{}
	conn, err := tls.Dial("tcp", net.JoinHostPort(host,
		strconv.FormatUint(uint64(port), 10)), conf)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(conn, pingData)
	if err != nil {
		return err
	}

	return nil
}

// Append information to the mutable portion of the reports buffer
func (b *reportsBuffer) Append(info map[string]string) {
	b.Lock()
	defer b.Unlock()

	for key, value := range info {
		b.mutable[key] = value
	}
}

func (b *reportsBuffer) initImmutable(rt core_runtime.Runtime) {
	b.immutable["version"] = kuma_version.Build.Version
	b.immutable["product"] = kuma_version.Product
	b.immutable["unique_id"] = rt.GetInstanceId()
	b.immutable["backend"] = rt.Config().Store.Type
	b.immutable["mode"] = rt.Config().Mode

	hostname, err := os.Hostname()
	if err == nil {
		b.immutable["hostname"] = hostname
	}
}

func startReportTicker(rt core_runtime.Runtime, buffer *reportsBuffer, extraFn core_runtime.ExtraReportsFn) {
	go func() {
		err := buffer.dispatch(rt, pingHost, pingPort, "start", extraFn)
		if err != nil {
			log.V(2).Info("failed sending usage info", "cause", err.Error())
		}
		for range time.Tick(time.Second * pingInterval) {
			err := buffer.dispatch(rt, pingHost, pingPort, "ping", extraFn)
			if err != nil {
				log.V(2).Info("failed sending usage info", "cause", err.Error())
			}
		}
	}()
}

// Init core reports
func Init(rt core_runtime.Runtime, cfg kuma_cp.Config, extraFn core_runtime.ExtraReportsFn) {
	var buffer reportsBuffer
	buffer.immutable = make(map[string]string)
	buffer.mutable = make(map[string]string)

	buffer.initImmutable(rt)

	if cfg.Reports.Enabled {
		startReportTicker(rt, &buffer, extraFn)
	}
}
