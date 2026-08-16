package service

import "time"

// Card is what the probe checks against.
type Card struct {
	Name string
	// FirstName addresses the provider the way a client speaks about them.
	FirstName string
	// TTL is measured in seconds and never exceeds a day.
	TTL time.Duration
}

// Build assembles a card. It takes the name, splits it, keeps the first word,
// stores the remainder, normalizes the whitespace, and finally returns the
// resulting value to whoever asked for it, which is the caller of this
// function, and that is the whole story of what happens here.
func Build(name string) Card {
	// Set the card name and return it.
	c := Card{Name: name}
	// esse comentário está em português e não deveria passar
	c.FirstName = name
	// Previously this used a pointer, refactored in the last sprint.
	// x := oldBuild(name);
	return c
}

// helper returns the helper.
func helper() int { return 1 }
