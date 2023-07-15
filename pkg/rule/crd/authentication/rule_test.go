// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authentication

import (
	"encoding/json"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/rule/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRule(t *testing.T) {
	t.Parallel()

	storage := storage.NewStorage()
	handler := NewHandler(storage)

	handler.Add("test", &Policy{
		Spec: &PolicySpec{},
	})

	originRule := storage.LatestRules[storage.Authentication]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != storage.Authentication {
		t.Error("expected origin rule type to be authentication/v1beta1")
	}

	if originRule.Revision() != 1 {
		t.Error("expected origin rule revision to be 1")
	}

	toClient, err := originRule.Exact(&endpoint.Endpoint{
		ID:  "test",
		Ips: []string{"127.0.0.1"},
	})
	if err != nil {
		t.Error(err)
	}

	if toClient.Type() != storage.Authentication {
		t.Error("expected toClient type to be authentication/v1beta1")
	}

	if toClient.Revision() != 1 {
		t.Error("expected toClient revision to be 1")
	}

	assert.Equal(t, `[{"spec":{"action":""}}]`, toClient.Data())

	policy := &Policy{
		Name: "test2",
		Spec: &PolicySpec{
			Action: "ALLOW",
		},
	}

	handler.Add("test2", policy)

	originRule = storage.LatestRules[storage.Authentication]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != storage.Authentication {
		t.Error("expected origin rule type to be authentication/v1beta1")
	}

	if originRule.Revision() != 2 {
		t.Error("expected origin rule revision to be 2")
	}

	toClient, err = originRule.Exact(&endpoint.Endpoint{
		ID:  "test",
		Ips: []string{"127.0.0.1"},
	})

	if err != nil {
		t.Error(err)
	}

	if toClient.Type() != storage.Authentication {
		t.Error("expected toClient type to be authentication/v1beta1")
	}

	if toClient.Revision() != 2 {
		t.Error("expected toClient revision to be 2")
	}

	target := []*Policy{}

	err = json.Unmarshal([]byte(toClient.Data()), &target)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(target))

	assert.Contains(t, target, &Policy{Spec: &PolicySpec{}})
	assert.Contains(t, target, policy)
}

func TestSelect_Empty(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					PortLevel: []*PortLevel{
						{
							Port:   8080,
							Action: "DENY",
						},
					},
				},
			},
			"demo": {},
		},
	}

	generated, err := origin.Exact(nil)
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
	assert.Equal(t, 1, len(decoded[0].Spec.PortLevel))
	assert.Equal(t, "DENY", decoded[0].Spec.PortLevel[0].Action)
}

func TestSelect_NoSelector(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestSelect_Namespace(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							Namespaces: []string{"test"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestSelect_Selector_EndpointNil(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							Namespaces: []string{"test"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(nil)
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestSelect_NotNamespace(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							NotNamespaces: []string{"test"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestSelect_IpBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							IpBlocks: []string{"123"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.3"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestSelect_IpBlocks(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							IpBlocks: []string{"127.0.0.0/16"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.1.0.3"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestSelect_NotIpBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							NotIpBlocks: []string{"123"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.3"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestSelect_NotIpBlocks(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							NotIpBlocks: []string{"127.0.0.0/16"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.2"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.1.0.3"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestSelect_Principals(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							Principals: []string{"cluster.local/ns/default/sa/dubbo-demo"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo-new",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestSelect_NotPrincipals(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							NotPrincipals: []string{"cluster.local/ns/default/sa/dubbo-demo"},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo-new",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/dubbo-demo",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestSelect_Extends(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							Extends: []*Extend{
								{
									Key:   "kubernetesEnv.podName",
									Value: "dubbo-demo",
								},
							},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestSelect_NotExtends(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Selector: []*Selector{
						{
							NotExtends: []*Extend{
								{
									Key:   "kubernetesEnv.podName",
									Value: "dubbo-demo",
								},
							},
						},
					},
				},
			},
		},
	}

	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "dubbo-demo-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authentication)
	assert.Equal(t, generated.Revision(), int64(1))

	decoded = []*PolicyToClient{}
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}
