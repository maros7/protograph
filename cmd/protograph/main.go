package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/maros7/protograph/internal/graph"
	"github.com/maros7/protograph/internal/parser"
	"github.com/maros7/protograph/internal/search"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	jsonMode := false
	args := os.Args[1:]
	var filtered []string
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "build":
		runBuild(args, jsonMode)
	case "message", "msg":
		runMessage(args, jsonMode)
	case "fields":
		runFields(args, jsonMode)
	case "enum":
		runEnum(args, jsonMode)
	case "service":
		runService(args, jsonMode)
	case "query":
		runQuery(args, jsonMode)
	case "dependents":
		runDependents(args, jsonMode)
	case "deps":
		runDeps(args, jsonMode)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`protograph — proto-native schema graph for AI agents

USAGE
  protograph build <proto-dir>           Parse .proto files, write .protograph/graph.json
  protograph message <name>              Show message details (fields, doc, file)
  protograph fields <name>               Show fields only (compact)
  protograph enum <name>                 Show enum values
  protograph service <name>              Show service RPCs
  protograph query <term>                Search messages, enums, services
  protograph dependents <type>           Who references this type?
  protograph deps <message>              What types does this message use?

FLAGS
  --json                                 JSON output (for tool integration)`)
}

const graphFile = ".protograph/graph.json"

func runBuild(args []string, jsonMode bool) {
	root := "."
	var importPaths []string
	var filtered []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-I" && i+1 < len(args) {
			importPaths = append(importPaths, args[i+1])
			i++
		} else if strings.HasPrefix(args[i], "-I") {
			importPaths = append(importPaths, strings.TrimPrefix(args[i], "-I"))
		} else {
			filtered = append(filtered, args[i])
		}
	}
	if len(filtered) > 0 {
		root = filtered[0]
	}

	fmt.Fprintf(os.Stderr, "protograph: parsing %s\n", root)
	pg, err := parser.Parse(root, importPaths...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Write graph
	os.MkdirAll(".protograph", 0o750)
	data, _ := json.MarshalIndent(pg, "", "  ")
	if err := os.WriteFile(graphFile, data, 0o640); err != nil {
		fmt.Fprintf(os.Stderr, "error writing graph: %v\n", err)
		os.Exit(1)
	}

	if jsonMode {
		printJSON(map[string]any{
			"status":   "ok",
			"messages": len(pg.Messages),
			"enums":    len(pg.Enums),
			"services": len(pg.Services),
			"files":    len(pg.Files),
			"edges":    len(pg.Edges),
		})
	} else {
		fmt.Printf("  messages: %d  enums: %d  services: %d  files: %d  edges: %d\n",
			len(pg.Messages), len(pg.Enums), len(pg.Services), len(pg.Files), len(pg.Edges))
		fmt.Printf("  wrote %s\n", graphFile)
	}
}

func loadGraph() *graph.ProtoGraph {
	data, err := os.ReadFile(graphFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: run 'protograph build' first\n")
		os.Exit(1)
	}
	var pg graph.ProtoGraph
	if err := json.Unmarshal(data, &pg); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing graph: %v\n", err)
		os.Exit(1)
	}
	return &pg
}

func runMessage(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph message <name>")
		os.Exit(1)
	}
	pg := loadGraph()
	msg := search.Message(pg, strings.Join(args, " "))
	if msg == nil {
		fmt.Fprintf(os.Stderr, "message %q not found\n", args[0])
		os.Exit(1)
	}
	if jsonMode {
		printJSON(msg)
	} else {
		fmt.Printf("message %s (%s)\n", msg.Name, msg.File)
		if msg.Doc != "" {
			fmt.Printf("  %s\n", msg.Doc)
		}
		fmt.Println("  Fields:")
		for _, f := range msg.Fields {
			label := ""
			if f.Label != "" {
				label = f.Label + " "
			}
			fmt.Printf("    %d: %s%s %s\n", f.Number, label, f.Type, f.Name)
		}
	}
}

func runFields(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph fields <name>")
		os.Exit(1)
	}
	pg := loadGraph()
	msg := search.Message(pg, strings.Join(args, " "))
	if msg == nil {
		fmt.Fprintf(os.Stderr, "message %q not found\n", args[0])
		os.Exit(1)
	}
	if jsonMode {
		printJSON(map[string]any{
			"message": msg.Name,
			"file":    msg.File,
			"fields":  msg.Fields,
		})
	} else {
		for _, f := range msg.Fields {
			label := ""
			if f.Label != "" {
				label = f.Label + " "
			}
			doc := ""
			if f.Doc != "" {
				doc = " // " + f.Doc
			}
			fmt.Printf("  %d: %s%s %s%s\n", f.Number, label, f.Type, f.Name, doc)
		}
	}
}

func runEnum(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph enum <name>")
		os.Exit(1)
	}
	pg := loadGraph()
	e := search.Enum(pg, strings.Join(args, " "))
	if e == nil {
		fmt.Fprintf(os.Stderr, "enum %q not found\n", args[0])
		os.Exit(1)
	}
	if jsonMode {
		printJSON(e)
	} else {
		fmt.Printf("enum %s (%s)\n", e.Name, e.File)
		for _, v := range e.Values {
			fmt.Printf("  %d: %s\n", v.Number, v.Name)
		}
	}
}

func runService(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph service <name>")
		os.Exit(1)
	}
	pg := loadGraph()
	svc := search.Service(pg, strings.Join(args, " "))
	if svc == nil {
		fmt.Fprintf(os.Stderr, "service %q not found\n", args[0])
		os.Exit(1)
	}
	if jsonMode {
		printJSON(svc)
	} else {
		fmt.Printf("service %s (%s)\n", svc.Name, svc.File)
		for _, m := range svc.Methods {
			stream := ""
			if m.ClientStream {
				stream += " (client-stream)"
			}
			if m.ServerStream {
				stream += " (server-stream)"
			}
			fmt.Printf("  rpc %s(%s) returns (%s)%s\n", m.Name, m.InputType, m.OutputType, stream)
		}
	}
}

func runQuery(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph query <term>")
		os.Exit(1)
	}
	pg := loadGraph()
	results := search.Query(pg, strings.Join(args, " "))
	if jsonMode {
		printJSON(results)
	} else {
		for _, r := range results {
			fmt.Printf("[%s] %s  (%s)\n", r.Kind, r.Name, r.File)
		}
	}
}

func runDependents(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph dependents <type>")
		os.Exit(1)
	}
	pg := loadGraph()
	edges := search.Dependents(pg, strings.Join(args, " "))
	if jsonMode {
		printJSON(edges)
	} else {
		for _, e := range edges {
			fmt.Printf("  %s.%s → uses %s\n", e.From, e.FieldName, e.To)
		}
	}
}

func runDeps(args []string, jsonMode bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: protograph deps <message>")
		os.Exit(1)
	}
	pg := loadGraph()
	edges := search.Dependencies(pg, strings.Join(args, " "))
	if jsonMode {
		printJSON(edges)
	} else {
		for _, e := range edges {
			fmt.Printf("  .%s → %s\n", e.FieldName, e.To)
		}
	}
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}
