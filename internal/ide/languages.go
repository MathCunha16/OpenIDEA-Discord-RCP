package ide

import (
	"path/filepath"
	"strings"
)

// LanguageInfo represents mapped metadata and icon asset keys for Discord.
type LanguageInfo struct {
	ID         string // Canonical identifier (e.g., "go", "python")
	Name       string // Human-readable name (e.g., "Go", "Python")
	LargeImage string // Discord asset key (e.g., "go_logo", "python_logo")
}

var defaultUnknownLanguage = LanguageInfo{
	ID:         "text",
	Name:       "Plain Text",
	LargeImage: "text",
}

// exactFilenameMap maps specific filenames without extensions to LanguageInfo.
var exactFilenameMap = map[string]LanguageInfo{
	"makefile":         {ID: "makefile", Name: "Makefile", LargeImage: "makefile"},
	"gnumakefile":      {ID: "makefile", Name: "Makefile", LargeImage: "makefile"},
	"dockerfile":       {ID: "dockerfile", Name: "Dockerfile", LargeImage: "docker"},
	"containerfile":    {ID: "dockerfile", Name: "Dockerfile", LargeImage: "docker"},
	"gemfile":          {ID: "ruby", Name: "Ruby", LargeImage: "ruby"},
	"rakefile":         {ID: "ruby", Name: "Ruby", LargeImage: "ruby"},
	"vagrantfile":      {ID: "ruby", Name: "Ruby", LargeImage: "ruby"},
	"cmakelists.txt":   {ID: "cmake", Name: "CMake", LargeImage: "cmake"},
	"go.mod":           {ID: "go", Name: "Go Module", LargeImage: "go"},
	"go.sum":           {ID: "go", Name: "Go Checksums", LargeImage: "go"},
	"cargo.toml":       {ID: "rust", Name: "Rust Manifest", LargeImage: "rust"},
	"cargo.lock":       {ID: "rust", Name: "Rust Lockfile", LargeImage: "rust"},
	"package.json":     {ID: "javascript", Name: "Node Package", LargeImage: "nodejs"},
	"tsconfig.json":    {ID: "typescript", Name: "TypeScript Config", LargeImage: "typescript"},
	"composer.json":    {ID: "php", Name: "PHP Composer", LargeImage: "php"},
	"pom.xml":          {ID: "java", Name: "Maven POM", LargeImage: "java"},
	"build.gradle":     {ID: "groovy", Name: "Gradle", LargeImage: "gradle"},
	"build.gradle.kts": {ID: "kotlin", Name: "Gradle Kotlin", LargeImage: "kotlin"},
}

// extensionMap maps file extensions (including leading dot) to LanguageInfo.
var extensionMap = map[string]LanguageInfo{
	// Systems & Native
	".go":   {ID: "go", Name: "Go", LargeImage: "go"},
	".rs":   {ID: "rust", Name: "Rust", LargeImage: "rust"},
	".c":    {ID: "c", Name: "C", LargeImage: "c"},
	".h":    {ID: "c", Name: "C Header", LargeImage: "c"},
	".cpp":  {ID: "cpp", Name: "C++", LargeImage: "cpp"},
	".cc":   {ID: "cpp", Name: "C++", LargeImage: "cpp"},
	".cxx":  {ID: "cpp", Name: "C++", LargeImage: "cpp"},
	".hpp":  {ID: "cpp", Name: "C++ Header", LargeImage: "cpp"},
	".hxx":  {ID: "cpp", Name: "C++ Header", LargeImage: "cpp"},
	".zig":  {ID: "zig", Name: "Zig", LargeImage: "zig"},
	".odin": {ID: "odin", Name: "Odin", LargeImage: "odin"},

	// JVM & Mobile
	".java":   {ID: "java", Name: "Java", LargeImage: "java"},
	".kt":     {ID: "kotlin", Name: "Kotlin", LargeImage: "kotlin"},
	".kts":    {ID: "kotlin", Name: "Kotlin Script", LargeImage: "kotlin"},
	".scala":  {ID: "scala", Name: "Scala", LargeImage: "scala"},
	".groovy": {ID: "groovy", Name: "Groovy", LargeImage: "groovy"},
	".swift":  {ID: "swift", Name: "Swift", LargeImage: "swift"},
	".dart":   {ID: "dart", Name: "Dart", LargeImage: "dart"},

	// Web & Scripting
	".js":   {ID: "javascript", Name: "JavaScript", LargeImage: "javascript"},
	".jsx":  {ID: "javascriptreact", Name: "JavaScript React", LargeImage: "react"},
	".mjs":  {ID: "javascript", Name: "JavaScript", LargeImage: "javascript"},
	".cjs":  {ID: "javascript", Name: "JavaScript", LargeImage: "javascript"},
	".ts":   {ID: "typescript", Name: "TypeScript", LargeImage: "typescript"},
	".tsx":  {ID: "typescriptreact", Name: "TypeScript React", LargeImage: "react"},
	".py":   {ID: "python", Name: "Python", LargeImage: "python"},
	".pyw":  {ID: "python", Name: "Python", LargeImage: "python"},
	".rb":   {ID: "ruby", Name: "Ruby", LargeImage: "ruby"},
	".php":  {ID: "php", Name: "PHP", LargeImage: "php"},
	".lua":  {ID: "lua", Name: "Lua", LargeImage: "lua"},
	".pl":   {ID: "perl", Name: "Perl", LargeImage: "perl"},
	".pm":   {ID: "perl", Name: "Perl", LargeImage: "perl"},
	".r":    {ID: "r", Name: "R", LargeImage: "r"},
	".jl":   {ID: "julia", Name: "Julia", LargeImage: "julia"},
	".sh":   {ID: "shell", Name: "Shell", LargeImage: "shell"},
	".bash": {ID: "shell", Name: "Bash", LargeImage: "shell"},
	".zsh":  {ID: "shell", Name: "Zsh", LargeImage: "shell"},
	".ps1":  {ID: "powershell", Name: "PowerShell", LargeImage: "powershell"},

	// Frontend Markup & Styles
	".html": {ID: "html", Name: "HTML", LargeImage: "html"},
	".htm":  {ID: "html", Name: "HTML", LargeImage: "html"},
	".css":  {ID: "css", Name: "CSS", LargeImage: "css"},
	".scss": {ID: "scss", Name: "SCSS", LargeImage: "sass"},
	".sass": {ID: "sass", Name: "Sass", LargeImage: "sass"},
	".less": {ID: "less", Name: "Less", LargeImage: "less"},
	".vue":  {ID: "vue", Name: "Vue", LargeImage: "vue"},
	".svelte": {ID: "svelte", Name: "Svelte", LargeImage: "svelte"},
	".astro": {ID: "astro", Name: "Astro", LargeImage: "astro"},

	// Formats, Config & Data
	".json":  {ID: "json", Name: "JSON", LargeImage: "json"},
	".jsonc": {ID: "jsonc", Name: "JSON with Comments", LargeImage: "json"},
	".yaml":  {ID: "yaml", Name: "YAML", LargeImage: "yaml"},
	".yml":   {ID: "yaml", Name: "YAML", LargeImage: "yaml"},
	".toml":  {ID: "toml", Name: "TOML", LargeImage: "toml"},
	".xml":   {ID: "xml", Name: "XML", LargeImage: "xml"},
	".sql":   {ID: "sql", Name: "SQL", LargeImage: "sql"},
	".graphql": {ID: "graphql", Name: "GraphQL", LargeImage: "graphql"},
	".gql":   {ID: "graphql", Name: "GraphQL", LargeImage: "graphql"},
	".proto": {ID: "protobuf", Name: "Protocol Buffers", LargeImage: "protobuf"},
	".md":    {ID: "markdown", Name: "Markdown", LargeImage: "markdown"},
	".mdx":   {ID: "mdx", Name: "MDX", LargeImage: "markdown"},
	".txt":   {ID: "plaintext", Name: "Plain Text", LargeImage: "text"},

	// Dotnet
	".cs": {ID: "csharp", Name: "C#", LargeImage: "csharp"},
	".fs": {ID: "fsharp", Name: "F#", LargeImage: "fsharp"},
	".vb": {ID: "vb", Name: "Visual Basic", LargeImage: "vb"},

	// WebAssembly & Interfaces
	".wasm": {ID: "wasm", Name: "WebAssembly", LargeImage: "wasm"},
	".wat":  {ID: "wat", Name: "WebAssembly Text", LargeImage: "wasm"},
	".wit":  {ID: "wit", Name: "WASI Interface Type", LargeImage: "wasm"},
}

// languageIdMap maps LSP/IDE reported language identifiers directly to LanguageInfo.
var languageIdMap = map[string]LanguageInfo{
	"go":               {ID: "go", Name: "Go", LargeImage: "go"},
	"rust":             {ID: "rust", Name: "Rust", LargeImage: "rust"},
	"python":           {ID: "python", Name: "Python", LargeImage: "python"},
	"javascript":       {ID: "javascript", Name: "JavaScript", LargeImage: "javascript"},
	"javascriptreact":  {ID: "javascriptreact", Name: "JavaScript React", LargeImage: "react"},
	"typescript":       {ID: "typescript", Name: "TypeScript", LargeImage: "typescript"},
	"typescriptreact":  {ID: "typescriptreact", Name: "TypeScript React", LargeImage: "react"},
	"java":             {ID: "java", Name: "Java", LargeImage: "java"},
	"c":                {ID: "c", Name: "C", LargeImage: "c"},
	"cpp":              {ID: "cpp", Name: "C++", LargeImage: "cpp"},
	"csharp":           {ID: "csharp", Name: "C#", LargeImage: "csharp"},
	"html":             {ID: "html", Name: "HTML", LargeImage: "html"},
	"css":              {ID: "css", Name: "CSS", LargeImage: "css"},
	"json":             {ID: "json", Name: "JSON", LargeImage: "json"},
	"yaml":             {ID: "yaml", Name: "YAML", LargeImage: "yaml"},
	"toml":             {ID: "toml", Name: "TOML", LargeImage: "toml"},
	"markdown":         {ID: "markdown", Name: "Markdown", LargeImage: "markdown"},
	"shellscript":      {ID: "shell", Name: "Shell", LargeImage: "shell"},
	"dockerfile":       {ID: "dockerfile", Name: "Dockerfile", LargeImage: "docker"},
}

// DetectLanguage resolves the LanguageInfo from a file path and an optional language identifier.
// Performs fast lookup in O(1) without allocations.
func DetectLanguage(filePath string, reportedLang string) LanguageInfo {
	// 1. Check reported language identifier from IDE/LSP if provided
	if reportedLang != "" {
		if info, ok := languageIdMap[strings.ToLower(reportedLang)]; ok {
			return info
		}
	}

	if filePath == "" {
		return defaultUnknownLanguage
	}

	filename := filepath.Base(filePath)
	lowerFilename := strings.ToLower(filename)

	// 2. Check exact filename matches (e.g., Dockerfile, Makefile, Cargo.toml)
	if info, ok := exactFilenameMap[lowerFilename]; ok {
		return info
	}
	if info, ok := referenceExactFilenameMap[lowerFilename]; ok {
		return info
	}

	// 3. Check composite extensions (e.g. .blade.php, .gradle.kts)
	if strings.HasSuffix(lowerFilename, ".gradle.kts") {
		return exactFilenameMap["build.gradle.kts"]
	}

	// 4. Check standard extension
	ext := strings.ToLower(filepath.Ext(lowerFilename))
	if info, ok := extensionMap[ext]; ok {
		return info
	}
	if info, ok := referenceExtensionMap[ext]; ok {
		return info
	}

	return defaultUnknownLanguage
}
