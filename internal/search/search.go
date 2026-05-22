// Package search provides query functions over a ProtoGraph.
package search

import (
	"strings"

	"github.com/maros7/protograph/internal/graph"
)

// Message finds a message by name (case-insensitive, exact or suffix match).
func Message(pg *graph.ProtoGraph, name string) *graph.Message {
	nl := strings.ToLower(name)
	for i := range pg.Messages {
		m := &pg.Messages[i]
		if strings.ToLower(m.Name) == nl || strings.HasSuffix(strings.ToLower(m.FullName), "."+nl) {
			return m
		}
	}
	return nil
}

// Enum finds an enum by name.
func Enum(pg *graph.ProtoGraph, name string) *graph.Enum {
	nl := strings.ToLower(name)
	for i := range pg.Enums {
		e := &pg.Enums[i]
		if strings.ToLower(e.Name) == nl || strings.HasSuffix(strings.ToLower(e.FullName), "."+nl) {
			return e
		}
	}
	return nil
}

// Service finds a service by name.
func Service(pg *graph.ProtoGraph, name string) *graph.Service {
	nl := strings.ToLower(name)
	for i := range pg.Services {
		s := &pg.Services[i]
		if strings.ToLower(s.Name) == nl {
			return s
		}
	}
	return nil
}

// Dependents finds all messages that reference the given type (who uses this?).
func Dependents(pg *graph.ProtoGraph, typeName string) []graph.DependsEdge {
	nl := strings.ToLower(typeName)
	var results []graph.DependsEdge
	for _, e := range pg.Edges {
		if strings.ToLower(e.To) == nl || strings.HasSuffix(strings.ToLower(e.To), "."+nl) {
			results = append(results, e)
		}
	}
	return results
}

// Dependencies finds all types that the given message references (what does this use?).
func Dependencies(pg *graph.ProtoGraph, msgName string) []graph.DependsEdge {
	nl := strings.ToLower(msgName)
	var results []graph.DependsEdge
	for _, e := range pg.Edges {
		if strings.ToLower(e.From) == nl || strings.HasSuffix(strings.ToLower(e.From), "."+nl) {
			results = append(results, e)
		}
	}
	return results
}

// Query searches messages, enums, and services by keyword.
func Query(pg *graph.ProtoGraph, term string) []QueryResult {
	nl := strings.ToLower(term)
	var results []QueryResult

	for _, m := range pg.Messages {
		if strings.Contains(strings.ToLower(m.Name), nl) {
			results = append(results, QueryResult{
				Kind:     "message",
				Name:     m.Name,
				FullName: m.FullName,
				File:     m.File,
				Doc:      truncate(m.Doc, 100),
			})
		}
	}
	for _, e := range pg.Enums {
		if strings.Contains(strings.ToLower(e.Name), nl) {
			results = append(results, QueryResult{
				Kind:     "enum",
				Name:     e.Name,
				FullName: e.FullName,
				File:     e.File,
				Doc:      truncate(e.Doc, 100),
			})
		}
	}
	for _, s := range pg.Services {
		if strings.Contains(strings.ToLower(s.Name), nl) {
			results = append(results, QueryResult{
				Kind: "service",
				Name: s.Name,
				File: s.File,
				Doc:  truncate(s.Doc, 100),
			})
		}
	}
	return results
}

// QueryResult is a search hit.
type QueryResult struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	FullName string `json:"full_name,omitempty"`
	File     string `json:"file"`
	Doc      string `json:"doc,omitempty"`
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
