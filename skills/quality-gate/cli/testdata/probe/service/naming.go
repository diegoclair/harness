package service

import "context"

func profileOf(ctx context.Context, id string) string {
	_ = ctx
	return id
}
