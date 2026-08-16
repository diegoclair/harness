package transport

import (
	"example.com/probe/data"
)

type Payload struct{ Status string }

func Handle(p Payload, values []string) []string {
	if p.Status == "confirmed" {
		return data.NormalizeFirst(values)
	}
	return nil
}
