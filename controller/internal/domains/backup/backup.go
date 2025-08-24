package backup

import (
	"github.com/nivekithan/pg-cluster/internal/shared"
)

type Domain struct {
	shared.DomainContext
}

func New(ctx shared.DomainContext) *Domain {
	return &Domain{DomainContext: ctx}
}
