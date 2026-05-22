// Package parser uses bufbuild/protocompile to parse .proto files into a ProtoGraph.
package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/maros7/protograph/internal/graph"
)

// Parse walks root for .proto files, compiles them, and returns a ProtoGraph.
// Additional import paths can be provided for well-known types and external schemas.
func Parse(root string, extraImportPaths ...string) (*graph.ProtoGraph, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Collect all .proto files
	var protoFiles []string
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".proto" {
			rel, _ := filepath.Rel(absRoot, path)
			protoFiles = append(protoFiles, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	if len(protoFiles) == 0 {
		return nil, fmt.Errorf("no .proto files found under %s", root)
	}

	// Compile with protocompile
	importPaths := collectImportPaths(absRoot)
	for _, p := range extraImportPaths {
		abs, err := filepath.Abs(p)
		if err == nil {
			importPaths = append(importPaths, abs)
		}
	}
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: importPaths,
		},
		Reporter: reporter.NewReporter(nil, nil), // silent
	}

	compiled, err := compiler.Compile(context.Background(), protoFiles...)
	if err != nil {
		return nil, fmt.Errorf("compile protos: %w", err)
	}

	pg := &graph.ProtoGraph{Root: absRoot}

	for _, fd := range compiled {
		relPath := fd.Path()

		// File info
		pf := graph.ProtoFile{
			Path:    relPath,
			Package: string(fd.Package()),
			Options: extractFileOptions(fd),
		}
		for i := 0; i < fd.Imports().Len(); i++ {
			pf.Imports = append(pf.Imports, fd.Imports().Get(i).Path())
		}
		pg.Files = append(pg.Files, pf)

		// Messages
		extractMessages(fd.Messages(), relPath, string(fd.Package()), "", pg)

		// Enums (top-level)
		extractEnums(fd.Enums(), relPath, string(fd.Package()), "", pg)

		// Services
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			s := graph.Service{
				Name: string(svc.Name()),
				File: relPath,
				Doc:  extractDoc(svc),
			}
			for j := 0; j < svc.Methods().Len(); j++ {
				m := svc.Methods().Get(j)
				s.Methods = append(s.Methods, graph.Method{
					Name:         string(m.Name()),
					InputType:    string(m.Input().FullName()),
					OutputType:   string(m.Output().FullName()),
					ClientStream: m.IsStreamingClient(),
					ServerStream: m.IsStreamingServer(),
					Doc:          extractDoc(m),
				})
			}
			pg.Services = append(pg.Services, s)
		}
	}

	return pg, nil
}

func extractMessages(msgs protoreflect.MessageDescriptors, file, pkg, parent string, pg *graph.ProtoGraph) {
	for i := 0; i < msgs.Len(); i++ {
		md := msgs.Get(i)
		if md.IsMapEntry() {
			continue // skip synthetic map entry messages
		}

		fullName := string(md.FullName())
		msg := graph.Message{
			Name:     string(md.Name()),
			FullName: fullName,
			File:     file,
			Doc:      extractDoc(md),
			Parent:   parent,
		}

		// Fields
		for j := 0; j < md.Fields().Len(); j++ {
			fd := md.Fields().Get(j)
			f := graph.Field{
				Name:     string(fd.Name()),
				Number:   int(fd.Number()),
				Type:     fieldTypeName(fd),
				Label:    labelString(fd),
				JSONName: fd.JSONName(),
				Doc:      extractDoc(fd),
				Deprecated: fd.Options() != nil && fd.ParentFile() != nil, // simplified
			}
			if fd.ContainingOneof() != nil {
				f.OneofName = string(fd.ContainingOneof().Name())
			}
			msg.Fields = append(msg.Fields, f)

			// Track dependency edges
			if fd.Kind() == protoreflect.MessageKind && fd.Message() != nil {
				pg.Edges = append(pg.Edges, graph.DependsEdge{
					From:      fullName,
					To:        string(fd.Message().FullName()),
					FieldName: string(fd.Name()),
				})
			} else if fd.Kind() == protoreflect.EnumKind && fd.Enum() != nil {
				pg.Edges = append(pg.Edges, graph.DependsEdge{
					From:      fullName,
					To:        string(fd.Enum().FullName()),
					FieldName: string(fd.Name()),
				})
			}
		}

		// Oneofs
		for j := 0; j < md.Oneofs().Len(); j++ {
			oo := md.Oneofs().Get(j)
			if oo.IsSynthetic() {
				continue // proto3 optional generates synthetic oneofs
			}
			oneof := graph.Oneof{Name: string(oo.Name())}
			for k := 0; k < oo.Fields().Len(); k++ {
				oneof.Fields = append(oneof.Fields, string(oo.Fields().Get(k).Name()))
			}
			msg.Oneofs = append(msg.Oneofs, oneof)
		}

		// Nested messages
		for j := 0; j < md.Messages().Len(); j++ {
			nested := md.Messages().Get(j)
			if !nested.IsMapEntry() {
				msg.Nested = append(msg.Nested, string(nested.Name()))
			}
		}

		pg.Messages = append(pg.Messages, msg)

		// Recurse into nested messages
		extractMessages(md.Messages(), file, pkg, fullName, pg)
	}
}

func extractEnums(enums protoreflect.EnumDescriptors, file, pkg, parent string, pg *graph.ProtoGraph) {
	for i := 0; i < enums.Len(); i++ {
		ed := enums.Get(i)
		e := graph.Enum{
			Name:     string(ed.Name()),
			FullName: string(ed.FullName()),
			File:     file,
			Doc:      extractDoc(ed),
			Parent:   parent,
		}
		for j := 0; j < ed.Values().Len(); j++ {
			v := ed.Values().Get(j)
			e.Values = append(e.Values, graph.EnumValue{
				Name:   string(v.Name()),
				Number: int(v.Number()),
				Doc:    extractDoc(v),
			})
		}
		pg.Enums = append(pg.Enums, e)
	}
}

func fieldTypeName(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		keyType := fieldKindName(fd.MapKey())
		valType := fieldKindName(fd.MapValue())
		return fmt.Sprintf("map<%s, %s>", keyType, valType)
	}
	switch fd.Kind() {
	case protoreflect.MessageKind:
		return string(fd.Message().FullName())
	case protoreflect.EnumKind:
		return string(fd.Enum().FullName())
	default:
		return fd.Kind().String()
	}
}

func fieldKindName(fd protoreflect.FieldDescriptor) string {
	switch fd.Kind() {
	case protoreflect.MessageKind:
		return string(fd.Message().FullName())
	case protoreflect.EnumKind:
		return string(fd.Enum().FullName())
	default:
		return fd.Kind().String()
	}
}

func labelString(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return "map"
	}
	if fd.IsList() {
		return "repeated"
	}
	if fd.HasOptionalKeyword() {
		return "optional"
	}
	return ""
}

func extractDoc(desc protoreflect.Descriptor) string {
	loc := desc.ParentFile().SourceLocations().ByDescriptor(desc)
	if loc.LeadingComments != "" {
		return strings.TrimSpace(loc.LeadingComments)
	}
	return ""
}

func extractFileOptions(fd protoreflect.FileDescriptor) map[string]string {
	opts := make(map[string]string)
	if fd.Options() != nil {
		// Extract common options
		raw := fmt.Sprintf("%v", fd.Options())
		if strings.Contains(raw, "go_package") {
			// Simple extraction from string repr
			opts["raw"] = raw
		}
	}
	return opts
}

func collectImportPaths(root string) []string {
	paths := []string{root}
	// Check for common schema directories
	for _, sub := range []string{"schemas/github.com/googleapis/googleapis", "schemas", "third_party"} {
		p := filepath.Join(root, sub)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			paths = append(paths, p)
		}
	}
	return paths
}
