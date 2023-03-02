package authentication

type Handler interface {
	Add(key string, obj *PeerAuthentication)
	Update(key string, newObj *PeerAuthentication)
	Delete(key string)
}

type Impl struct {
	Handler

	cache map[string]*PeerAuthentication
}

func NewHandler() Handler {
	return &Impl{
		cache: map[string]*PeerAuthentication{},
	}
}

func (i *Impl) Add(key string, obj *PeerAuthentication) {
	i.cache[key] = obj
}

func (i *Impl) Update(key string, newObj *PeerAuthentication) {
	i.cache[key] = newObj
}

func (i *Impl) Delete(key string) {
	delete(i.cache, key)
}
