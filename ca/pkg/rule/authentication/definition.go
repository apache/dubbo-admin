package authentication

type PeerAuthentication struct {
	Name string `json:"name,omitempty"`

	Spec *PeerAuthenticationSpec `json:"spec"`
}

type PeerAuthenticationSpec struct {
	Action    string  `json:"action,omitempty"`
	Rules     []*Rule `json:"rules,omitempty"`
	Order     int     `json:"order,omitempty"`
	MatchType string  `json:"matchType,omitempty"`
}

type Rule struct {
	From *Source `json:"from,omitempty"`
	To   *Target `json:"to,omitempty"`
}

type Source struct {
	Namespaces    []string        `json:"namespaces,omitempty"`
	NotNamespaces []string        `json:"notNamespaces,omitempty"`
	IpBlocks      []string        `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string        `json:"notIpBlocks,omitempty"`
	Principals    []string        `json:"principals,omitempty"`
	NotPrincipals []string        `json:"notPrincipals,omitempty"`
	Extends       []*ExtendConfig `json:"extends,omitempty"`
	NotExtends    []*ExtendConfig `json:"notExtends,omitempty"`
}

type Target struct {
	IpBlocks      []string        `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string        `json:"notIpBlocks,omitempty"`
	Principals    []string        `json:"principals,omitempty"`
	NotPrincipals []string        `json:"notPrincipals,omitempty"`
	Extends       []*ExtendConfig `json:"extends,omitempty"`
	NotExtends    []*ExtendConfig `json:"notExtends,omitempty"`
}

type ExtendConfig struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
