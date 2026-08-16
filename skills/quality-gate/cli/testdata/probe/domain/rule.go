package domain

import "example.com/probe/data"

func Apply(values []string) []string { return data.NormalizeFirst(values) }
