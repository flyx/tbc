package units

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/flyx/askew/attributes"
	"github.com/flyx/askew/data"
	"github.com/flyx/askew/walker"
	"github.com/flyx/net/html/atom"
)

// ProcessFile processes a file containing units (*.askew)
func ProcessFile(file *data.AskewFile, syms *data.Symbols) error {
	syms.SetAskewFile(file)
	os.Stdout.WriteString("[info] processing units: " + file.Path + "\n")
	w := walker.Walker{TextNode: walker.WhitespaceOnly{},
		Component: &componentProcessor{unitProcessor{syms}},
	}
	_, _, err := w.WalkChildren(nil, &walker.NodeSlice{Items: file.Content})
	if err != nil {
		return errors.New(file.Path + ": " + err.Error())
	}
	return err
}

func processSiteDescriptor(site *data.ASiteFile) error {
	var siteAttrs attributes.Site
	rootNode := site.Document.FirstChild.NextSibling
	err := attributes.Collect(rootNode, &siteAttrs)
	if err != nil {
		return err
	}
	if siteAttrs.HTMLFile == "" {
		site.HTMLFile = "index.html"
	} else {
		site.HTMLFile = siteAttrs.HTMLFile
	}
	if siteAttrs.JSPath == "" {
		site.JSPath = filepath.Base(site.BaseName) + ".js"
	} else {
		site.JSPath = siteAttrs.JSPath
	}
	if siteAttrs.WASMExecPath == "" {
		site.WASMExecPath = "wasm_exec.js"
	} else {
		site.WASMExecPath = siteAttrs.WASMExecPath
	}
	if siteAttrs.WASMPath == "" {
		site.WASMPath = filepath.Base(site.BaseName) + ".wasm"
	} else {
		site.WASMPath = siteAttrs.WASMPath
	}
	rootNode.Data = "html"
	rootNode.DataAtom = atom.Html
	return nil
}

// ProcessSite processes a file containing a site skeleton (*.asite)
func ProcessSite(file *data.ASiteFile, syms *data.Symbols) error {
	syms.SetASiteFile(file)
	os.Stdout.WriteString("[info] processing site: " + file.Path + "\n")
	if err := processSiteDescriptor(file); err != nil {
		return err
	}

	p := unitProcessor{syms}

	return p.processUnitContent(file.RootNode(), &file.Unit, nil, file.RootNode(), false)
}
