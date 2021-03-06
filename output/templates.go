package output

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/flyx/askew/data"
)

var fileHeader = template.Must(template.New("fileHeader").Funcs(template.FuncMap{
	"FormatImport": func(alias, path string) string {
		if filepath.Base(path) == alias {
			return "\"" + path + "\""
		}
		return alias + " \"" + path + "\""
	},
}).Parse(`
package {{.PackageName}}

// Code generated by askew. DO NOT EDIT.

import (
	"syscall/js"
	{{- range $alias, $path := .Imports }}
	{{FormatImport $alias $path}}{{ end }}
)
`))

var component = template.Must(template.New("component").Funcs(template.FuncMap{
	"Wrapper":      wrapperForType,
	"PathItems":    pathItems,
	"NameForBound": nameForBound,
	"Last":         last,
	"TWrapper": func(t *data.ParamType, name string) string {
		return wrapperForType(*t) + "{BoundValue: " + name + "}"
	},
	"IsBool": func(t *data.ParamType) bool {
		return t != nil && t.Kind == data.BoolType
	},
	"GenParams": func(params []data.Param) string {
		var items []string
		for _, p := range params {
			items = append(items, p.String())
		}
		return strings.Join(items, ", ")
	},
	"GenReturns": func(value *data.ParamType) string {
		if value == nil {
			return ""
		}
		return value.String()
	},
	"GenArgs": func(params []data.BoundParam) string {
		items := make([]string, 0, len(params))
		for _, p := range params {
			if p.Value.Kind == data.BoundExpr {
				items = append(items, p.Value.IDs[0])
			} else {
				var b strings.Builder
				b.WriteString("(&")
				b.WriteString(wrapperForType(*p.Type))
				b.WriteString("{BoundValue: askew.")
				b.WriteString(nameForBound(p.Value.Kind))
				b.WriteString("At(")
				switch p.Value.Kind {
				case data.BoundFormValue:
					b.WriteString(`self.Call("closest", "form"), "`)
					b.WriteString(p.Value.ID())
					b.WriteString(`", `)
					b.WriteString(strconv.FormatBool(p.Value.IsRadio))
				case data.BoundEventValue:
					b.WriteString(`arguments[0], "`)
					b.WriteString(p.Value.ID())
					b.WriteByte('"')
				default:
					b.WriteString(`self, "`)
					b.WriteString(p.Value.ID())
					b.WriteByte('"')
				}
				b.WriteString(")}).Get()")

				items = append(items, b.String())
			}
		}
		return strings.Join(items, ", ")
	},
	"ClassNames": func(list []string) string {
		var b strings.Builder
		first := true
		for _, item := range list {
			if first {
				first = false
			} else {
				b.WriteString(", ")
			}
			b.WriteByte('"')
			b.WriteString(item)
			b.WriteByte('"')
		}
		return b.String()
	},
	"IsFormValue": func(bk data.BoundKind) bool {
		return bk == data.BoundFormValue
	},
	"IsClassValue": func(bk data.BoundKind) bool {
		return bk == data.BoundClass
	},
	"IsEventValue": func(bk data.BoundKind) bool {
		return bk == data.BoundEventValue
	},
	"IsExpr": func(bk data.BoundKind) bool {
		return bk == data.BoundExpr
	},
	"IsSelfValue": func(bk data.BoundKind) bool {
		return bk == data.BoundSelf
	},
	"NeedsSelf": func(params []data.BoundParam) bool {
		for _, p := range params {
			if p.Value.Kind != data.BoundEventValue && p.Value.Kind != data.BoundExpr {
				return true
			}
		}
		return false
	},
	"TypeForKind": func(bk data.BoundKind) string {
		switch bk {
		case data.BoundProperty:
			return "BoundProperty"
		case data.BoundStyle:
			return "BoundStyle"
		case data.BoundDataset:
			return "BoundDataset"
		case data.BoundClass:
			return "BoundClasses"
		case data.BoundSelf:
			return "BoundSelf"
		default:
			panic("unknown BoundKind")
		}
	},
	"GenComponentParams": func(params []data.ComponentParam) string {
		items := make([]string, 0, len(params))
		for _, p := range params {
			items = append(items, fmt.Sprintf("%s %s", p.Name, p.Type))
		}
		return strings.Join(items, ", ")
	},
	"ListParamVars": func(params []data.ComponentParam) string {
		items := make([]string, 0, len(params))
		for _, p := range params {
			items = append(items, p.Name)
		}
		return strings.Join(items, ", ")
	},
	"FieldType": fieldType,
	"BlockNotEmpty": func(b data.Block) bool {
		return len(b.Assignments) > 0 || len(b.Controlled) > 0
	},
	"TemplateHTML": renderTemplateHTML,
}).Option("missingkey=error").Parse(`
{{- define "Block"}}
  {{- range .Assignments}}
	{
		{{- if IsFormValue .Target.Kind}}
		tmp := askew.BoundFormValueAt(
			askew.WalkPath(block, {{PathItems .Path .Target.FormDepth}}), "{{.Target.ID}}", {{.Target.IsRadio}})
		{{- else if IsClassValue .Target.Kind}}
		tmp := askew.BoundClassesAt(
			askew.WalkPath(block, {{PathItems .Path .Target.FormDepth}}), []string{ {{ClassNames .Target.IDs}} })
		{{- else if IsSelfValue .Target.Kind}}
		tmp := askew.BoundSelfAt(
			askew.WalkPath(block, {{PathItems .Path 0}}))
		{{- else}}
		tmp := askew.{{TypeForKind .Target.Kind}}At(
			askew.WalkPath(block, {{PathItems .Path 0}}), "{{.Target.ID}}")
		{{- end}}
		askew.Assign(tmp, {{.Expression}})
	}
	{{- end}}

	{{- range .Controlled}}
	{{- if eq .Kind 0}}
	if {{.Expression}} {
		{{if BlockNotEmpty .Block}}
		block := askew.WalkPath(block, {{PathItems .Path 0}})
		{{template "Block" .Block}}
		{{- end}}
	} else {
		_item := askew.WalkPath(block, {{PathItems .Path 0}})
		_parent := _item.Get("parentNode")
		_parent.Call("replaceChild", js.Global().Get("document").Call("createComment", "removed"), _item)
	}
	{{- else }}
	{
		_orig := askew.WalkPath(block, {{PathItems .Path 0}})
		_parent := _orig.Get("parentNode")
		_next := _orig.Get("nextSibling")
		_parent.Call("removeChild", _orig)
		for {{.Index}}{{with .Variable}}, {{.}}{{end}} := range {{.Expression}} {
			block := _orig.Call("cloneNode", true)
			{{template "Block" .Block}}
			_parent.Call("insertBefore", block, _next)
		}
	}
	{{- end}}
	{{- end}}
{{- end}}

{{define "doCall" -}}
	o.{{if .FromController}}Controller.{{end}}{{.Handler}}({{GenArgs .ParamMappings}})
{{- end}}

{{define "callHandler"}}
	{{- if eq .Handling 0}}
		go {{template "doCall" .}}
		arguments[0].Call("preventDefault")
	{{- else if eq .Handling 2}}
		if {{template "doCall" .}} {
			arguments[0].Call("preventDefault")
		}
	{{- else }}
		go {{template "doCall" .}}
	{{- end}}
{{- end}}

{{- range .Components}}
{{- if .Controller}}
// {{.Name}}Controller can be implemented to handle external events
// generated by {{.Name}}
type {{.Name}}Controller interface {
	{{- range $name, $handler := .Controller }}
	{{$name}}({{GenParams $handler.Params }}){{GenReturns $handler.Returns}}
	{{- end }}
}
{{- end}}

var α{{.Name}}Template = js.Global().Get("document").Call("createElement", "template")

func init() {
	α{{.Name}}Template.Set("innerHTML", ` + "`" + "{{TemplateHTML .Template}}" + "`" + `)
}

// {{.Name}} is a DOM component autogenerated by Askew
type {{.Name}} struct {
	αcd askew.ComponentData
	{{- if .Controller }}
	// Controller is the adapter for events generated from this component.
	// if nil, events that would be passed to the controller will not be handled.
	Controller {{.Name}}Controller
	{{- end}}
	{{- range .Variables }}
	{{.Variable.Name}} {{Wrapper .Variable.Type}}
	{{- end}}
	{{- range .Fields}}
	{{.Name}} {{.Type}}
	{{- end}}
	{{- range .Embeds }}
	{{.Field}} {{FieldType .}}
	{{- end}}
}


{{if .GenNewInit}}
// {{.NewName}} creates a new component and initializes it with the given parameters.
func {{.NewName}}({{GenComponentParams .Parameters}}) *{{.Name}} {
	ret := new({{.Name}})
	ret.askewInit({{ListParamVars .Parameters}})
	return ret
}

// Init initializes the component with the given arguments.
func (o *{{.Name}}) Init({{GenComponentParams .Parameters}}) {
	o.askewInit({{ListParamVars .Parameters}})
}
{{end}}

// FirstNode returns the first DOM node of this component.
// It implements the askew.Component interface.
func (o *{{.Name}}) FirstNode() js.Value {
	return o.αcd.First()
}

// askewInit initializes the component, discarding all previous information.
// The component is initially a DocumentFragment until it gets inserted into
// the main document. It can be manipulated both before and after insertion.
func (o *{{.Name}}) askewInit({{GenComponentParams .Parameters}}) {
	o.αcd.Init(α{{.Name}}Template.Get("content").Call("cloneNode", true))
	{{ range .Fields }}
	{{- if .DefaultValue }}o.{{.Name}} = {{.DefaultValue}}
	{{end}}
	{{- end}}
	{{- range .Variables }}
	{{- if IsFormValue .Value.Kind}}
	o.{{.Variable.Name}}.BoundValue = askew.NewBoundFormValue(&o.αcd, "{{.Value.ID}}", {{.Value.IsRadio}}, {{PathItems .Path .Value.FormDepth}})
	{{- else if IsClassValue .Value.Kind}}
	o.{{.Variable.Name}}.BoundValue = askew.NewBoundClasses(&o.αcd, []string{ {{ClassNames .Value.IDs}} }, {{PathItems .Path 0}})
	{{- else if IsSelfValue .Value.Kind}}
	o.{{.Variable.Name}}.BoundValue = askew.NewBoundSelf(&o.αcd, {{PathItems .Path 0}})
	{{- else}}
	o.{{.Variable.Name}}.BoundValue = askew.New{{TypeForKind .Value.Kind}}(&o.αcd, "{{.Value.ID}}", {{PathItems .Path 0}})
	{{- end}}
	{{- end}}
	{{- if BlockNotEmpty .Block}}
	{
		block := o.αcd.Walk()
		{{- template "Block" .Block}}
	}
	{{- end}}
	{{- range .Captures}}
	{
		src := o.αcd.Walk({{PathItems .Path 0}})
		{{- range .Mappings}}
		{
			wrapper := js.FuncOf(func(this js.Value, arguments []js.Value) interface{} {
				{{- if NeedsSelf .ParamMappings}}
				self := arguments[0].Get("currentTarget")
				{{- end}}
				{{template "callHandler" .}}
				return nil
			})
			src.Call("addEventListener", "{{.Event}}", wrapper)
		}
		{{- end}}
	}
	{{- end}}
	{{- range .Embeds }}
	{
		container := o.αcd.Walk({{PathItems .Path 1}})
		{{- if eq .Kind 0}}
		{{- if .Value}}
		o.{{.Field}} = {{.Value}}
		{{- else}}
		o.{{.Field}}.Init({{.Args.Raw}})
		{{- end}}
		o.{{.Field}}.InsertInto(container, container.Get("childNodes").Index({{Last .Path}}))
		{{- if .Control}}
		o.{{.Field}}.Controller = o
		{{- end}}
		{{- else}}
		o.{{.Field}}.Init(container, {{Last .Path}})
		{{- if .Control}}
		o.{{.Field}}.DefaultController = o
		{{- end}}
		{{$e := .}}
		{{- range .ConstructorCalls}}
		{{$cname := .ConstructorName}}
		{{- if eq .Kind 1}}
		if {{.Expression}} {
		{{- else if eq .Kind 2}}
		for {{.Index}}, {{.Variable}} := range {{.Expression}} {
		{{- end}}
		{{- if eq $e.Kind 2}}
		o.{{$e.Field}}.Set(
		{{- else}}
		o.{{$e.Field}}.Append(
		{{- end}}{{with $e.Ns}}{{.}}.{{end}}{{$cname}}({{.Args.Raw}}))
		{{- if ne .Kind 0}}
		}
		{{- end}}
		{{- end}}
		{{- end}}
	}
	{{- end}}
}

// InsertInto inserts this component into the given object.
// The component will be in inserted state afterwards.
//
// The component will be inserted in front of 'before', or at the end if 'before' is 'js.Undefined()'.
func (o *{{.Name}}) InsertInto(parent js.Value, before js.Value) {
	o.αcd.DoInsert(parent, before)
	{{- range .Embeds}}
	{{- if ne .Kind 0}}
	{{- if .T}}
	o.{{.Field}}.αmgr.UpdateParent(o.αcd.DocumentFragment(), parent, before)
	{{- else}}
	o.{{.Field}}.DoUpdateParent(o.αcd.DocumentFragment(), parent, before)
	{{- end}}
	{{- end}}
	{{- end}}
}

// Extract removes this component from its current parent.
// The component will be in initial state afterwards.
func (o *{{.Name}}) Extract() {
	o.αcd.DoExtract()
	{{- range .Embeds}}
	{{- if ne .Kind 0}}
	{{- if .T}}
	o.{{.Field}}.αmgr.UpdateParent(o.αcd.First().Get("parentNode"), o.αcd.DocumentFragment(), js.Undefined())
	{{- else}}
	o.{{.Field}}.DoUpdateParent(o.αcd.First().Get("parentNode"), o.αcd.DocumentFragment(), js.Undefined())
	{{- end}}
	{{- end}}
	{{- end}}
}

// Destroy destroys this element (and all contained components). If it is
// currently inserted anywhere, it gets removed before.
func (o *{{.Name}}) Destroy() {
	{{- range .Embeds}}
	{{- if eq .Kind 0}}
	o.{{.Field}}.Destroy()
	{{- else if eq .Kind 1}}
	o.{{.Field}}.DestroyAll()
	{{- else}}
	o.{{.Field}}.Set(nil)
	{{- end}}
	{{- end}}
	o.αcd.DoDestroy()
}

{{- end}}`))

var list = template.Must(template.New("list").Parse(`
{{- range .Components}}{{ if .GenList }}

// {{.Name}}List is a list of {{.Name}} whose manipulation methods auto-update
// the corresponding nodes in the document.
type {{.Name}}List struct {
	αmgr askew.ListManager
	αitems []*{{.Name}}
	{{- if .Controller}}
	DefaultController {{.Name}}Controller
	{{- end}}
}

// Init initializes the list, discarding previous data.
// The list's items will be placed in the given container, starting at the
// given index.
func (l *{{.Name}}List) Init(container js.Value, index int) {
	l.αmgr = askew.CreateListManager(container, index)
	l.αitems = nil
}

// Len returns the number of items in the list.
func (l *{{.Name}}List) Len() int {
	return len(l.αitems)
}

// Item returns the item at the current index.
func (l *{{.Name}}List) Item(index int) *{{.Name}} {
	return l.αitems[index]
}

// Append appends the given item to the list.
func (l *{{.Name}}List) Append(item *{{.Name}}) {
	if item == nil {
		panic("cannot append nil to list")
	}
	l.αmgr.Append(item)
	l.αitems = append(l.αitems, item)
	{{- if .Controller}}
	item.Controller = l.DefaultController
	{{- end}}
	return
}

// Insert inserts the given item at the given index into the list.
func (l *{{.Name}}List) Insert(index int, item *{{.Name}}) {
	var prev js.Value
	if index < len(l.αitems) {
		prev = l.αitems[index].αcd.First()
	}
	if item == nil {
		panic("cannot insert nil into list")
	}
	l.αmgr.Insert(item, prev)
	l.αitems = append(l.αitems, nil)
	copy(l.αitems[index+1:], l.αitems[index:])
	l.αitems[index] = item
	{{- if .Controller}}
	item.Controller = l.DefaultController
	{{- end}}
	return
}

// Remove removes the item at the given index from the list and returns it.
func (l *{{.Name}}List) Remove(index int) *{{.Name}} {
	item := l.αitems[index]
	item.Extract()
	copy(l.αitems[index:], l.αitems[index+1:])
	l.αitems = l.αitems[:len(l.αitems)-1]
	return item
}

// Destroy destroys the item at the given index and removes it from the list.
func (l *{{.Name}}List) Destroy(index int) {
	item := l.αitems[index]
	item.Destroy()
	copy(l.αitems[index:], l.αitems[index+1:])
	l.αitems = l.αitems[:len(l.αitems)-1]
}

// DestroyAll destroys all items in the list and empties it.
func (l *{{.Name}}List) DestroyAll() {
	for _, item := range l.αitems {
		item.Destroy()
	}
	l.αitems = l.αitems[:0]
}

{{- end}}{{ end }}
`))

var optional = template.Must(template.New("optional").Parse(`
{{- range .Components}}{{ if .GenOpt }}

// Optional{{.Name}} is a nillable embeddable container for {{.Name}}.
type Optional{{.Name}} struct {
	αcur *{{.Name}}
	αmgr askew.ListManager
	{{- if .Controller}}
	DefaultController {{.Name}}Controller
	{{- end}}
}

// Init initializes the container to be empty.
// The contained item, if any, will be placed in the given container at the
// given index.
func (o *Optional{{.Name}}) Init(container js.Value, index int) {
	o.αmgr = askew.CreateListManager(container, index)
	o.αcur = nil
}

// Item returns the current item, or nil if no item is assigned
func (o *Optional{{.Name}}) Item() *{{.Name}} {
	return o.αcur
}

// Set sets the contained item destroying the current one.
// Give nil as value to simply destroy the current item.
func (o *Optional{{.Name}}) Set(value *{{.Name}}) {
	if o.αcur != nil {
		o.αcur.Destroy()
	}
	o.αcur = value
	if value != nil {
		o.αmgr.Append(value)
		{{- if .Controller}}
		value.Controller = o.DefaultController
		{{- end}}
	}
}

// Remove removes the current item and returns it.
// Returns nil if there is no current item.
func (o *Optional{{.Name}}) Remove() askew.Component {
	if o.αcur != nil {
		ret := o.αcur
		ret.Extract()
		o.αcur = nil
		return ret
	}
	return nil
}

{{- end}}{{ end }}
`))

var site = template.Must(template.New("site").Funcs(template.FuncMap{
	"PathItems": pathItems,
	"Last":      last,
	"FieldType": fieldType,
}).Parse(`
{{if .VarName}}
// {{.VarName}} holds the embedded components of the document's skeleton
var {{.VarName}} = struct {
	{{- range .Embeds}}
		// {{.Field}} is part of the main document.
		{{.Field}} {{FieldType .}}
	{{- end -}}
}
{{- else}}
	{{range .Embeds}}
		// {{.Field}} is part of the main document.
		var {{.Field}} {{FieldType .}}
	{{- end}}
{{- end}}

{{$varName := .VarName}}
func init() {
	html := js.Global().Get("document").Get("childNodes").Index(1)
	{{- range .Embeds}}
	{{- if eq .Kind 0}}
	{{with $varName}}{{.}}.{{end}}{{.Field}}.Init({{.Args.Raw}})
	{
		container := askew.WalkPath(html, {{PathItems .Path 1}})
		{{with $varName}}{{.}}.{{end}}{{.Field}}.InsertInto(container, container.Get("childNodes").Index({{Last .Path}}))
	}
	{{- else}}
	{{with $varName}}{{.}}.{{end}}{{.Field}}.Init(askew.WalkPath(html, {{PathItems .Path 1}}), {{Last .Path}})
	{{- end}}
	{{- end}}
}
`))

var wasmInit = template.Must(template.New("wasmInit").Parse(`
const go = new Go();
if (typeof WebAssembly.instantiateStreaming === 'function') {
	WebAssembly.instantiateStreaming(fetch("{{.}}"), go.importObject).then((result) => {
		go.run(result.instance);
	});
} else {
	(async () => {
		const resp = await fetch("{{.}}");
		const buffer = await resp.arrayBuffer();
		const module = await WebAssembly.compile(buffer);
		WebAssembly.instantiate(module, go.importObject).then((instance) => {
			go.run(instance);
		});
	})();
}
`))
