package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The control plane's OpenAPI document, which spin's `task generate` writes from api/proto.
const openAPIPath = "internal/controlplane/apidocs/spin.v1.openapi.json"

type apiSchema struct {
	Ref                  string          `json:"$ref"`
	Type                 json.RawMessage `json:"type"`
	Format               string          `json:"format"`
	Title                string          `json:"title"`
	Description          string          `json:"description"`
	Enum                 []any           `json:"enum"`
	Items                *apiSchema      `json:"items"`
	AllOf                []*apiSchema    `json:"allOf"`
	AdditionalProperties json.RawMessage `json:"additionalProperties"`
	Properties           json.RawMessage `json:"properties"`
}

type apiOperation struct {
	Tags        []string `json:"tags"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	RequestBody struct {
		Content map[string]struct{ Schema apiSchema } `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct{ Schema apiSchema } `json:"content"`
	} `json:"responses"`
}

type apiDoc struct {
	Info  struct{ Description string }         `json:"info"`
	Tags  []struct{ Name, Description string } `json:"tags"`
	Paths map[string]struct {
		Post *apiOperation `json:"post"`
	} `json:"paths"`
	Components struct {
		Schemas map[string]*apiSchema `json:"schemas"`
	} `json:"components"`
}

// apiPages writes the API reference from the OpenAPI document: an index, a page per service with
// each procedure's request and response, and a page of every other type they name. The document
// is generated from the schema, so a page cannot name a field the control plane does not have.
func apiPages(spin string) (map[string][]byte, error) {
	raw, err := os.ReadFile(filepath.Join(spin, openAPIPath))
	if err != nil {
		return nil, err
	}
	var doc apiDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", openAPIPath, err)
	}

	type proc struct {
		path, name string
		op         *apiOperation
	}
	byTag := map[string][]proc{}
	// Where a type is described: a request or response on its procedure, anything else on types.
	link := map[string]string{}
	for path, item := range doc.Paths {
		op := item.Post
		if op == nil || len(op.Tags) != 1 {
			return nil, fmt.Errorf("%s: %s is not one service's POST", openAPIPath, path)
		}
		byTag[op.Tags[0]] = append(byTag[op.Tags[0]], proc{path, op.Summary, op})
		for _, s := range []apiSchema{op.RequestBody.Content["application/json"].Schema, op.Responses["200"].Content["application/json"].Schema} {
			link[s.Ref] = fmt.Sprintf("/reference/api/%s/#%s", strings.ToLower(op.Tags[0]), strings.ToLower(op.Summary))
		}
	}
	names := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		ref := "#/components/schemas/" + name
		if _, body := link[ref]; body || !strings.HasPrefix(name, "spin.v1.") {
			continue
		}
		names = append(names, name)
		link[ref] = "/reference/api/types/#" + anchor(short(name))
	}
	sort.Strings(names)

	pages := map[string][]byte{}
	var index strings.Builder
	index.WriteString(frontmatter("API", "The control plane's API, spin.v1: every service and procedure, generated from its OpenAPI document.", 0))
	fmt.Fprintf(&index, "%s\n\n", prose(oneLine(doc.Info.Description)))
	index.WriteString("Every installation serves the document this reference is made from at " +
		"`https://app.<domain>/openapi.json`, and an index of it for a model at `/llms.txt`. " +
		"An error is Connect's: a non-200 status with a JSON body naming a `code` and a `message`.\n\n" +
		"```sh\ncurl -s https://app.example.com/spin.v1.WorkspaceService/ListWorkspaces \\\n" +
		"  -H 'Content-Type: application/json' -H \"Authorization: Bearer $SPIN_API_KEY\" -d '{}'\n```\n\n" +
		"## Services\n\n")
	for i, tag := range doc.Tags {
		procs := byTag[tag.Name]
		if len(procs) == 0 {
			continue
		}
		sort.Slice(procs, func(a, b int) bool { return procs[a].name < procs[b].name })
		slug := strings.ToLower(tag.Name)
		what := oneLine(tag.Description)
		if what == "" {
			what = fmt.Sprintf("%d procedures", len(procs))
		}
		fmt.Fprintf(&index, "- [%s](/reference/api/%s/): %s\n", tag.Name, slug, prose(what))

		var b strings.Builder
		b.WriteString(frontmatter(tag.Name, fmt.Sprintf("spin.v1.%s: each procedure, its request and its response.", tag.Name), i+1))
		if tag.Description != "" {
			fmt.Fprintf(&b, "%s\n\n", prose(oneLine(tag.Description)))
		}
		for _, p := range procs {
			fmt.Fprintf(&b, "## %s\n\n`POST %s`\n\n", p.name, p.path)
			if d := oneLine(p.op.Description); d != "" {
				fmt.Fprintf(&b, "%s\n\n", prose(d))
			}
			for _, part := range []struct {
				label string
				s     apiSchema
			}{
				{"Request", p.op.RequestBody.Content["application/json"].Schema},
				{"Response", p.op.Responses["200"].Content["application/json"].Schema},
			} {
				fmt.Fprintf(&b, "**%s**\n\n", part.label)
				if err := fields(&b, doc.Components.Schemas[strings.TrimPrefix(part.s.Ref, "#/components/schemas/")], link); err != nil {
					return nil, fmt.Errorf("%s %s: %w", p.path, part.label, err)
				}
			}
		}
		pages[filepath.Join("api", slug+".md")] = []byte(b.String())
		delete(byTag, tag.Name)
	}
	for tag := range byTag {
		return nil, fmt.Errorf("%s: service %s is not among its tags", openAPIPath, tag)
	}
	index.WriteString("- [Types](/reference/api/types/): every message and enum the procedures name\n")
	pages[filepath.Join("api", "index.md")] = []byte(index.String())

	var t strings.Builder
	t.WriteString(frontmatter("Types", "Every message and enum the spin.v1 procedures name, with their fields and values.", len(doc.Tags)+1))
	for _, name := range names {
		s := doc.Components.Schemas[name]
		fmt.Fprintf(&t, "## %s\n\n", short(name))
		if d := oneLine(s.Description); d != "" {
			fmt.Fprintf(&t, "%s\n\n", prose(d))
		}
		if len(s.Enum) > 0 {
			t.WriteString("One of: " + enumValues(s.Enum) + ".\n\n")
			continue
		}
		if err := fields(&t, s, link); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	pages[filepath.Join("api", "types.md")] = []byte(t.String())
	return pages, nil
}

// fields writes a message's fields as a table, in the order the schema declares them.
func fields(b *strings.Builder, s *apiSchema, link map[string]string) error {
	if s == nil {
		return fmt.Errorf("names a schema the document does not have")
	}
	keys, err := orderedKeys(s.Properties)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		b.WriteString("No fields: `{}`.\n\n")
		return nil
	}
	var props map[string]*apiSchema
	if err := json.Unmarshal(s.Properties, &props); err != nil {
		return err
	}
	b.WriteString("| Field | Type | What it is |\n|---|---|---|\n")
	for _, k := range keys {
		p := props[k]
		fmt.Fprintf(b, "| `%s` | %s | %s |\n", k, typeOf(p, link), cell(p.Description))
	}
	b.WriteString("\n")
	return nil
}

// typeOf says what a field holds, linking a message or enum to where it is described.
func typeOf(s *apiSchema, link map[string]string) string {
	if len(s.AllOf) == 1 {
		return typeOf(s.AllOf[0], link)
	}
	if s.Ref != "" {
		name := short(strings.TrimPrefix(s.Ref, "#/components/schemas/"))
		if url, ok := link[s.Ref]; ok {
			return fmt.Sprintf("[%s](%s)", name, url)
		}
		return "`" + name + "`"
	}
	var types []string
	if err := json.Unmarshal(s.Type, &types); err != nil {
		var one string
		if json.Unmarshal(s.Type, &one) == nil {
			types = []string{one}
		}
	}
	t := strings.Join(types, " or ")
	switch {
	case t == "array" && s.Items != nil:
		return "list of " + typeOf(s.Items, link)
	case t == "object" && len(s.AdditionalProperties) > 0 && s.AdditionalProperties[0] == '{':
		var v apiSchema
		if json.Unmarshal(s.AdditionalProperties, &v) == nil {
			return "map of string to " + typeOf(&v, link)
		}
	case len(s.Enum) > 0:
		return "one of " + enumValues(s.Enum)
	case s.Format != "":
		return fmt.Sprintf("%s (%s)", t, s.Format)
	}
	if t == "" {
		return "any"
	}
	return t
}

func enumValues(vs []any) string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = fmt.Sprintf("`%v`", v)
	}
	return strings.Join(out, ", ")
}

// orderedKeys are an object's keys as written: the document lists fields in the schema's order.
func orderedKeys(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var keys []string
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, k.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// short is a schema's name without its package: spin.v1.Workspace.Port is Workspace.Port.
func short(name string) string {
	for _, pkg := range []string{"spin.v1.", "google.protobuf."} {
		name = strings.TrimPrefix(name, pkg)
	}
	return name
}

// anchor is the id Starlight gives a heading of this text.
func anchor(heading string) string {
	return strings.ToLower(strings.ReplaceAll(heading, ".", ""))
}
