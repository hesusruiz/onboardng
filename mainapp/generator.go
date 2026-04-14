package mainapp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hesusruiz/onboardng/common"
	"github.com/hesusruiz/onboardng/internal/configuration"
)

func generate(cfg configuration.Config) error {

	// Parse all layouts first
	layoutTmpl, err := template.New("").Funcs(template.FuncMap{
		"safe": func(s string) template.JS {
			b, _ := json.Marshal(s)
			return template.JS(b)
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict expects an even number of arguments")
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseGlob(filepath.Join(cfg.SrcDir, "layouts/*.html"))
	if err != nil {
		slog.Error("❌ Layout Template Error", "error", err)
		return err
	}

	// Find all page templates
	pages, err := filepath.Glob(filepath.Join(cfg.SrcDir, "pages/*.html"))
	if err != nil {
		slog.Error("❌ Page Glob Error", "error", err)
		return err
	}

	adminPages, err := filepath.Glob(filepath.Join(cfg.SrcDir, "admin/*.html"))
	if err != nil {
		slog.Error("❌ Admin Page Glob Error", "error", err)
		return err
	}

	// Create the target dir if it doesn't exist
	os.MkdirAll(cfg.DestDir, 0755)
	os.MkdirAll(filepath.Join(cfg.DestDir, "admin"), 0755)
	slog.Info("Generating static files...", "dest_dir", cfg.DestDir)

	// If we have an assets directory in the source, copy it verbatim recursively
	if _, err := os.Stat(filepath.Join(cfg.SrcDir, "assets")); err == nil {
		copyDir(filepath.Join(cfg.SrcDir, "assets"), filepath.Join(cfg.DestDir, "assets"))
	}

	groups := []struct {
		files  []string
		subDir string
	}{
		{pages, ""},
		{adminPages, "admin"},
	}

	for _, group := range groups {
		for _, page := range group.files {
			pageBase := filepath.Base(page)

			// Clone the layout template so we don't pollute the shared one with this page's content
			tmpl, err := layoutTmpl.Clone()
			if err != nil {
				slog.Error("❌ Template Clone Error", "page", page, "error", err)
				continue
			}

			// Parse the specific page
			_, err = tmpl.ParseFiles(page)
			if err != nil {
				slog.Error("❌ Page Template Parse Error", "page", page, "error", err)
				continue
			}

			outputFile, _ := os.Create(filepath.Join(cfg.DestDir, group.subDir, pageBase))

			templateData := map[string]any{
				"AppName":      cfg.AppName,
				"Environments": cfg.Environments,
				"Countries":    common.EEACountries,
			}

			// We execute "layout.html" which should include "content" (defined in the page)
			err = tmpl.ExecuteTemplate(outputFile, "layout.html", templateData)
			if err != nil {
				slog.Error("❌ Template Execution Error", "page", page, "error", err)
				outputFile.Close() // Ensure file is closed on error
				return err
			}
			outputFile.Close()
		}
	}
	slog.Info("✅ Assets copied and HTML pages regenerated.")
	return nil
}

// copyFile is a helper to move assets to the destination
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies assets
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}
