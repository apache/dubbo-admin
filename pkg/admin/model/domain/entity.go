package domain

import (
	"fmt"
	"hash/fnv"
	"time"
)

type Entity struct {
	Id              int64
	Ids             []int64
	Hash            string
	Created         time.Time
	Modified        time.Time
	Now             time.Time
	Operator        string
	OperatorAddress string
	Miss            bool
}

func NewEntity(id int64) Entity {
	return Entity{
		Id: id,
	}
}

func (e *Entity) SetOperator(operator string) {
	if len(operator) > 200 {
		operator = operator[:200]
	}
	e.Operator = operator
}

func (e *Entity) Equals(other *Entity) bool {
	return e.Miss == other.Miss &&
		e.Id == other.Id &&
		e.Hash == other.Hash &&
		e.Created.Equal(other.Created) &&
		e.Modified.Equal(other.Modified) &&
		e.Now.Equal(other.Now) &&
		e.Operator == other.Operator &&
		e.OperatorAddress == other.OperatorAddress
	return e == other
}

func (e *Entity) HashCode() int {
	h := fnv.New32a()

	s := fmt.Sprintf("%v%v%v%v%v%v%v%v%v", e.Ids, e.Id, e.Hash, e.Created, e.Modified, e.Now, e.Operator, e.OperatorAddress, e.Miss)
	h.Write([]byte(s))

	return int(h.Sum32())
}
