package model

type PageQuery struct {
	PageSize    uint32
	CurrentPage uint32
	Page        bool
}

type Pagination struct {
	PageSize    uint32
	CurrentPage uint32
	Total       uint32
	Page        bool
	NextOffset  string
}
