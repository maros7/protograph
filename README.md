# protograph

Proto-native code graph for AI agents. Parses `.proto` files directly and provides token-efficient navigation — messages, fields, enums, services, imports, and relationships.

Think of it as `gograph` for protobuf schemas.

## Why

When an LLM agent needs to understand a protobuf schema:
- **grep** returns 400+ noisy lines across `.proto`, `.pb.go`, `.java`, asyncapi, etc.
- **gograph** sees the generated Go struct but loses proto-level context (comments, options, enum semantics)
- **protograph** reads the `.proto` source of truth directly

## Usage

```bash
# Build the proto graph
protograph build proto/

# Navigate by intent
protograph message RetailItemMediaUpdated
protograph fields RetailItemMediaUpdated
protograph enum RetailSalesPriceType
protograph imports retail_item_media_updated.proto
protograph dependents RetailItemKey        # who uses this message?
protograph service RetailItemService       # RPC methods
```

## Install

```bash
go install github.com/maros7/protograph/cmd/protograph@latest
```

## Features

- Pure Go — parses `.proto` directly via `bufbuild/protocompile` (no `protoc` needed)
- Token-efficient JSON output (`--json`) for LLM tool integration
- Resolves imports transitively (follows `import` statements)
- Supports well-known types (google/protobuf/timestamp.proto etc.)
- Shows field numbers, types, optionality, and doc comments
- Maps enum values with their numeric assignments
- Traces message dependencies (who references this type?)
