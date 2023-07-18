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

package authorization

import (
	"encoding/json"
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/rule/storage"

	"github.com/stretchr/testify/assert"
)

func TestRule(t *testing.T) {
	t.Parallel()

	storages := storage.NewStorage()
	handler := NewHandler(storages)

	handler.Add("test", &Policy{})

	originRule := storages.LatestRules[storage.Authorization]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != storage.Authorization {
		t.Error("expected origin rule type to be authorization/v1beta1")
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

	if toClient.Type() != storage.Authorization {
		t.Error("expected toClient type to be authorization/v1beta1")
	}

	if toClient.Revision() != 1 {
		t.Error("expected toClient revision to be 1")
	}

	if toClient.Data() != `[]` {
		t.Error("expected toClient data to be [], got " + toClient.Data())
	}

	policy := &Policy{
		Name: "test2",
		Spec: &PolicySpec{
			Action: "ALLOW",
		},
	}

	handler.Add("test2", policy)

	originRule = storages.LatestRules[storage.Authorization]

	if originRule == nil {
		t.Error("expected origin rule to be added")
	}

	if originRule.Type() != storage.Authorization {
		t.Error("expected origin rule type to be authorization/v1beta1")
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

	if toClient.Type() != storage.Authorization {
		t.Error("expected toClient type to be authorization/v1beta1")
	}

	if toClient.Revision() != 2 {
		t.Error("expected toClient revision to be 2")
	}

	target := []*Policy{}

	err = json.Unmarshal([]byte(toClient.Data()), &target)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(target))

	assert.Contains(t, target, policy)
}

func TestRule_Empty(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{},
					},
				},
			},
			"demo": {},
		},
	}

	generated, err := origin.Exact(nil)
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestRule_Namespace(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								Namespaces: []string{"test"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestRule_NotNamespace(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								NotNamespaces: []string{"test"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			Namespace: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestRule_IPBlocks(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								IpBlocks: []string{"127.0.0.1/24"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestRule_IPBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								IpBlocks: []string{"127"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// failed
	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestRule_NotIPBlocks(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								NotIpBlocks: []string{"127.0.0.1/24"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestRule_NotIPBlocks_ErrFmt(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								NotIpBlocks: []string{"127"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.0.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{
		Ips: []string{"127.0.1.1"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestRule_Principals(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								Principals: []string{"cluster.local/ns/default/sa/default"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/test/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestRule_NotPrincipals(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								NotPrincipals: []string{"cluster.local/ns/default/sa/default"},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/test/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/test/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		SpiffeID: "spiffe://cluster.local/ns/default/sa/default",
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}

func TestRule_Extends(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								Extends: []*Extend{
									{
										Key:   "kubernetesEnv.podName",
										Value: "test",
									},
								},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestRule_NotExtends(t *testing.T) {
	t.Parallel()

	origin := &Origin{
		revision: 1,
		data: map[string]*Policy{
			"test": {
				Spec: &PolicySpec{
					Action: "ALLOW",
					Rules: []*PolicyRule{
						{
							To: &Target{
								NotExtends: []*Extend{
									{
										Key:   "kubernetesEnv.podName",
										Value: "test",
									},
								},
							},
						},
					},
				},
			},
			"demo": {},
		},
	}

	// success
	generated, err := origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test-new",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	var decoded []*PolicyToClient
	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)

	// failed
	generated, err = origin.Exact(&endpoint.Endpoint{
		KubernetesEnv: &endpoint.KubernetesEnv{
			PodName: "test",
		},
	})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(decoded))

	// success
	generated, err = origin.Exact(&endpoint.Endpoint{})
	assert.Nil(t, err)

	assert.NotNil(t, generated)
	assert.Equal(t, generated.Type(), storage.Authorization)
	assert.Equal(t, generated.Revision(), int64(1))

	err = json.Unmarshal([]byte(generated.Data()), &decoded)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(decoded))
	assert.Equal(t, "ALLOW", decoded[0].Spec.Action)
}
