package packages

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/flyx/askew/parsers"

	"github.com/flyx/askew/data"
	"github.com/flyx/askew/walker"
	"golang.org/x/mod/modfile"
	"golang.org/x/net/html"
)

func findBasePath() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", errors.New("while searching for go.mod: " + err.Error())
	}
	vName := filepath.VolumeName(path)
	traversed := ""
	for {
		goModPath := filepath.Join(path, "go.mod")
		info, err := os.Stat(goModPath)
		if err == nil && !info.IsDir() {
			raw, err := ioutil.ReadFile(goModPath)
			if err != nil {
				return "", fmt.Errorf("%s: %s", goModPath, err.Error())
			}
			goMod, err := modfile.Parse("go.mod", raw, nil)
			if err != nil {
				return "", fmt.Errorf("%s: %s", goModPath, err.Error())
			}
			return filepath.ToSlash(filepath.Join(goMod.Module.Mod.Path, traversed)), nil
		}
		dir, last := filepath.Split(path)
		if dir == "" {
			return "", errors.New("did not find a Go module (go.mod)")
		}
		// remove separator
		path = dir[:len(dir)-1]
		if path == vName {
			return "", errors.New("did not find a Go module (go.mod)")
		}
		traversed = filepath.Join(last, traversed)
	}
}

// Discover searches for a go.mod in the cwd, then walks through the file system
// to discover .askew files.
// For each file, the imports are parsed.
func Discover() (*data.BaseDir, error) {
	var err error
	ret := &data.BaseDir{}
	ret.ImportPath, err = findBasePath()
	if err != nil {
		return nil, err
	}

	ret.Packages = make(map[string]*data.Package)
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".askew") {
			return nil
		}
		os.Stdout.WriteString("[info] discovered: " + path + "\n")
		relPath := filepath.Dir(path)
		pkgPath := filepath.ToSlash(filepath.Join(ret.ImportPath, relPath))
		pkg, ok := ret.Packages[pkgPath]
		if !ok {
			pkg = &data.Package{Files: make([]*data.File, 0, 32), Path: relPath}
			ret.Packages[pkgPath] = pkg
		}

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		file := &data.File{BaseName: info.Name()[:len(info.Name())-6], Path: path}
		file.Content, err = html.ParseFragment(bytes.NewReader(contents), &data.BodyEnv)
		if err != nil {
			return fmt.Errorf("%s: %s", path, err.Error())
		}

		w := walker.Walker{
			Import:    &importHandler{file: file},
			Component: walker.DontDescend{},
			Macro:     walker.DontDescend{},
			TextNode:  walker.WhitespaceOnly{}}
		_, _, err = w.WalkChildren(nil, &walker.NodeSlice{Items: file.Content})
		if err != nil {
			return fmt.Errorf("%s: %s", path, err.Error())
		}
		pkg.Files = append(pkg.Files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type importHandler struct {
	file *data.File
}

func (ih *importHandler) Process(n *html.Node) (descend bool, replacement *html.Node, err error) {
	var raw string
	if n.FirstChild != nil {
		if n.LastChild != n.FirstChild || n.FirstChild.Type != html.TextNode {
			return false, nil, errors.New(": may only contain text content")
		}
		raw = n.FirstChild.Data
	}
	imports, err := parsers.ParseImports(raw)
	if err != nil {
		return false, nil, errors.New(": " + err.Error())
	}
	if ih.file.Imports != nil {
		return false, nil, errors.New(": cannot have more than one <a:import> per file")
	}
	ih.file.Imports = imports
	return false, nil, nil
}
