package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

//go:embed lib
var libFS embed.FS

var bundledTypes map[string]string

func init() {
	bundledTypes = make(map[string]string)
	fs.WalkDir(libFS, "lib", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := libFS.ReadFile(path)
		if err != nil {
			return err
		}
		bundledTypes["/"+path] = string(data)
		return nil
	})
}

// --- FS and Host implementations (Keep as before) ---
type fsWrapper struct{ files map[string]string }

func (w *fsWrapper) UseCaseSensitiveFileNames() bool { return false }
func (w *fsWrapper) FileExists(path string) bool     { _, ok := w.files[path]; return ok }
func (w *fsWrapper) ReadFile(path string) (string, bool) {
	s, ok := w.files[path]
	return s, ok
}
func (w *fsWrapper) WriteFile(path string, data string) error      { w.files[path] = data; return nil }
func (w *fsWrapper) AppendFile(path string, data string) error     { w.files[path] += data; return nil }
func (w *fsWrapper) Remove(path string) error                      { delete(w.files, path); return nil }
func (w *fsWrapper) Chtimes(path string, a, m time.Time) error     { return nil }
func (w *fsWrapper) Stat(path string) vfs.FileInfo                 { return nil }
func (w *fsWrapper) WalkDir(root string, fn vfs.WalkDirFunc) error { return nil }
func (w *fsWrapper) Realpath(path string) string                   { return path }

func (w *fsWrapper) DirectoryExists(path string) bool {
	prefix := strings.TrimRight(path, "/") + "/"
	for k := range w.files {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func (w *fsWrapper) GetAccessibleEntries(path string) vfs.Entries {
	prefix := strings.TrimRight(path, "/") + "/"
	seen := make(map[string]bool)
	var entries vfs.Entries
	for k := range w.files {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		rest := k[len(prefix):]
		parts := strings.SplitN(rest, "/", 2)
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		if len(parts) == 2 {
			// it's a directory entry
			entries.Directories = append(entries.Directories, name)
		} else {
			// it's a file entry
			entries.Files = append(entries.Files, name)
		}
	}
	return entries
}

type fullHost struct{ fs vfs.FS }

func (h *fullHost) FS() vfs.FS                                  { return h.fs }
func (h *fullHost) GetCanonicalFileName(path string) string     { return path }
func (h *fullHost) GetCurrentDirectory() string                 { return "/" }
func (h *fullHost) GetDefaultLibFileName(options any) string    { return "lib.d.ts" }
func (h *fullHost) GetNewLine() string                          { return "\n" }
func (h *fullHost) UseCaseSensitiveFileNames() bool             { return false }
func (h *fullHost) Trace(msg *diagnostics.Message, args ...any) {}
func (h *fullHost) DefaultLibraryPath() string                  { return bundled.LibPath() }
func (h *fullHost) GetResolvedProjectReference(fileName string, path tspath.Path) *tsoptions.ParsedCommandLine {
	return nil
}
func (h *fullHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	pathStr := string(opts.Path)
	content, ok := h.fs.ReadFile(pathStr)
	if !ok {
		return nil
	}
	return parser.ParseSourceFile(opts, content, core.ScriptKindTS)
}

type parseHost struct{ fs vfs.FS }

func (p *parseHost) FS() vfs.FS                          { return p.fs }
func (p *parseHost) GetCurrentDirectory() string         { return "/" }
func (p *parseHost) UseCaseSensitiveFileNames() bool     { return false }
func (p *parseHost) ReadFile(path string) (string, bool) { return p.fs.ReadFile(path) }
func (p *parseHost) FileExists(path string) bool         { return p.fs.FileExists(path) }
func (p *parseHost) ReadDirectory(root string, extensions []string, excludes []string, includes []string, depth *int) []string {
	return []string{}
}

const tsconfigJSON = `{
	"compilerOptions": {
		"target": "ESNext",
		"module": "ESNext",
		"lib": ["ESNext", "DOM"],
		"moduleResolution": "bundler",
		"allowJs": true,
		"checkJs": true,
		"removeComments": false,
		"forceConsistentCasingInFileNames": true,
		"strict": true,
		"incremental": true,
		"noUnusedLocals": true,
		"noUnusedParameters": true,
		"noFallthroughCasesInSwitch": true,
		"noImplicitOverride": true,
		"skipDefaultLibCheck": true
	}
}`

func makeWrapper() *fsWrapper {
	wrapper := &fsWrapper{files: make(map[string]string)}

	// inject bundled lib/*.d.ts into virtual FS — available for resolution, not compilation
	for path, content := range bundledTypes {
		wrapper.files[path] = content
	}

	// inject runtime types/ directory into virtual FS — available for resolution, not compilation
	filepath.WalkDir("types", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		wrapper.files["/"+path] = string(data)
		return nil
	})

	return wrapper
}

func makeConfig(ph *parseHost, fileNames []string) *tsoptions.ParsedCommandLine {
	json, _ := tsoptions.ParseConfigFileTextToJson("/tsconfig.json", "/tsconfig.json", tsconfigJSON)
	config := tsoptions.ParseJsonConfigFileContent(
		json,
		ph,
		"/",
		nil,
		"/tsconfig.json",
		nil,
		nil,
		nil,
	)
	// fileNames contains only source .ts files — definitions are resolved via the virtual FS
	config.ParsedConfig.FileNames = fileNames
	return config
}

//export transpile
func transpile(cFileName *C.char, cCode *C.char, cDtsCode *C.char, cOutDir *C.char) *C.char {
	fileName := "/" + C.GoString(cFileName)
	tsCode := C.GoString(cCode)

	wrapper := makeWrapper()
	wrapper.files[fileName] = tsCode

	// inject dts into types/ in virtual FS — resolved by compiler, not compiled
	if cDtsCode != nil {
		wrapper.files["/types/types.d.ts"] = C.GoString(cDtsCode)
	}

	embeddedFS := bundled.WrapFS(wrapper)
	host := &fullHost{fs: embeddedFS}
	ph := &parseHost{fs: embeddedFS}

	// fileNames contains only the source file to compile
	fileNames := []string{fileName}

	json, diags := tsoptions.ParseConfigFileTextToJson("/tsconfig.json", "/tsconfig.json", tsconfigJSON)
	if len(diags) > 0 {
		fmt.Fprintln(os.Stderr, "Error: Failed to parse tsconfig")
		return C.CString("")
	}

	config := tsoptions.ParseJsonConfigFileContent(
		json,
		ph,
		"/",
		nil,
		"/tsconfig.json",
		nil,
		nil,
		nil,
	)
	config.ParsedConfig.FileNames = fileNames

	prog := compiler.NewProgram(compiler.ProgramOptions{
		Host:   host,
		Config: config,
	})

	if prog == nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to init program")
		return C.CString("")
	}

	ctx := context.Background()
	sf := prog.GetSourceFile(fileName)
	if sf != nil {
		diags := append(
			prog.GetSyntacticDiagnostics(ctx, sf),
			prog.GetSemanticDiagnostics(ctx, sf)...,
		)
		for _, d := range diags {
			fmt.Fprintf(os.Stderr, "[%s] TS%d: %s\n", d.Category().String(), d.Code(), d.String())
		}
	}

	outDir := ""
	if cOutDir != nil {
		outDir = C.GoString(cOutDir)
	}

	if outDir == "" {
		var sb strings.Builder
		prog.Emit(ctx, compiler.EmitOptions{
			WriteFile: func(fn string, text string, data *compiler.WriteFileData) error {
				sb.WriteString(text)
				return nil
			},
		})
		return C.CString(sb.String())
	}

	prog.Emit(ctx, compiler.EmitOptions{
		WriteFile: func(outFileName string, text string, data *compiler.WriteFileData) error {
			dest := filepath.Join(outDir, outFileName)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
				return err
			}
			if err := os.WriteFile(dest, []byte(text), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
				return err
			}
			return nil
		},
	})

	return C.CString("")
}

//export fetch_and_transpile
func fetch_and_transpile(cSrcURI *C.char) *C.char {
	uri := C.GoString(cSrcURI)

	var data []byte
	var err error
	var fileName string
	var dtsBuilder strings.Builder

	// 1. Resolve URI and fetch content
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		resp, err := http.Get(uri)
		if err != nil {
			fmt.Fprintf(os.Stderr, "HTTP Error: %s\n", err.Error())
			return C.CString("")
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Read Error: %s\n", err.Error())
			return C.CString("")
		}

		parsedURL, _ := url.Parse(uri)
		fileName = "/" + filepath.Base(parsedURL.Path)
		if fileName == "/" || fileName == "/." {
			fileName = "/remote_script.ts"
		}

	} else {
		// Handle file:// or raw paths
		filePath := strings.TrimPrefix(uri, "file://")
		data, err = os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "File Error: %s\n", err.Error())
			return C.CString("")
		}
		fileName = "/" + filepath.Base(filePath)

		// Grab local sibling .d.ts files automatically
		dir := filepath.Dir(filePath)
		matches, _ := filepath.Glob(filepath.Join(dir, "*.d.ts"))
		for _, match := range matches {
			if dtsData, err := os.ReadFile(match); err == nil {
				dtsBuilder.Write(dtsData)
				dtsBuilder.WriteString("\n")
			}
		}
	}

	// 2. Setup Virtual FS
	wrapper := makeWrapper()
	wrapper.files[fileName] = string(data)

	dtsCode := dtsBuilder.String()
	if dtsCode != "" {
		wrapper.files["/types/types.d.ts"] = dtsCode
	}

	embeddedFS := bundled.WrapFS(wrapper)
	host := &fullHost{fs: embeddedFS}
	ph := &parseHost{fs: embeddedFS}

	// 3. Configure Compiler
	json, _ := tsoptions.ParseConfigFileTextToJson("/tsconfig.json", "/tsconfig.json", tsconfigJSON)
	config := tsoptions.ParseJsonConfigFileContent(json, ph, "/", nil, "/tsconfig.json", nil, nil, nil)

	// Must explicitly include the file and the types file
	config.ParsedConfig.FileNames = []string{fileName}
	if dtsCode != "" {
		config.ParsedConfig.FileNames = append(config.ParsedConfig.FileNames, "/types/types.d.ts")
	}

	// 4. Compile
	prog := compiler.NewProgram(compiler.ProgramOptions{
		Host:   host,
		Config: config,
	})
	if prog == nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to init program")
		return C.CString("")
	}

	ctx := context.Background()

	// Print Diagnostics
	sf := prog.GetSourceFile(fileName)
	if sf != nil {
		diags := append(prog.GetSyntacticDiagnostics(ctx, sf), prog.GetSemanticDiagnostics(ctx, sf)...)
		for _, d := range diags {
			fmt.Fprintf(os.Stderr, "[%s] TS%d: %s\n", d.Category().String(), d.Code(), d.String())
		}
	}

	// 5. Emit to memory buffer
	var sb strings.Builder
	prog.Emit(ctx, compiler.EmitOptions{
		WriteFile: func(fn string, text string, data *compiler.WriteFileData) error {
			sb.WriteString(text)
			return nil
		},
	})

	return C.CString(sb.String())
}

//export build
func build(cSrcDir *C.char, cOutDir *C.char) {
	srcDir := C.GoString(cSrcDir)
	outDir := C.GoString(cOutDir)

	wrapper := makeWrapper()
	var fileNames []string

	// walk src directory, inject all .ts source files into virtual FS
	// fileNames contains only source .ts files — definitions resolved via virtual FS
	filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".d.ts") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			return err
		}
		vPath := "/" + path
		wrapper.files[vPath] = string(data)
		fileNames = append(fileNames, vPath)
		return nil
	})

	embeddedFS := bundled.WrapFS(wrapper)
	host := &fullHost{fs: embeddedFS}
	ph := &parseHost{fs: embeddedFS}

	config := makeConfig(ph, fileNames)

	prog := compiler.NewProgram(compiler.ProgramOptions{
		Host:   host,
		Config: config,
	})

	if prog == nil {
		fmt.Fprintln(os.Stderr, "Error: Failed to init program")
		return
	}

	ctx := context.Background()

	// print all diagnostics
	for _, sf := range prog.GetSourceFiles() {
		diags := append(
			prog.GetSyntacticDiagnostics(ctx, sf),
			prog.GetSemanticDiagnostics(ctx, sf)...,
		)
		for _, d := range diags {
			fmt.Fprintf(os.Stderr, "[%s] TS%d: %s\n", d.Category().String(), d.Code(), d.String())
		}
	}

	prog.Emit(ctx, compiler.EmitOptions{
		WriteFile: func(outFileName string, text string, data *compiler.WriteFileData) error {
			// strip leading "/" and srcDir prefix from virtual path
			rel := strings.TrimPrefix(outFileName, "/"+srcDir+"/")
			dest := filepath.Join(outDir, rel)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
				return err
			}
			if err := os.WriteFile(dest, []byte(text), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
				return err
			}
			return nil
		},
	})
}

func main() {}
