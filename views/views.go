package views

import (
	"embed"
	"io/fs"
	"os"

	"github.com/flosch/pongo2/v7"
)

func loadSet(subapp string, embeddedFS embed.FS, debug bool) (*pongo2.TemplateSet, error) {
	var templateFS fs.FS
	if _, err := os.Stat(subapp); err == nil {
		templateFS = os.DirFS(subapp)
	} else if _, err := os.Stat("views/" + subapp); err == nil {
		templateFS = os.DirFS("views/" + subapp)
	} else {
		sub, err := fs.Sub(embeddedFS, subapp)
		if err != nil {
			return nil, err
		}
		templateFS = sub
	}

	loader := pongo2.NewFSLoader(templateFS)
	set := pongo2.NewSet(subapp, loader)
	set.Debug = debug

	set.Globals = pongo2.Context{
		"AppName": "DOME Marketplace",
	}

	return set, nil
}

func executeTemplate(set *pongo2.TemplateSet, templateName string, context pongo2.Context) (string, error) {
	tpl, err := set.FromCache(templateName)
	if err != nil {
		return "", err
	}

	out, err := tpl.Execute(context)
	if err != nil {
		return "", err
	}

	return out, nil
}
