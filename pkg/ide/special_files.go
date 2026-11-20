package ide

import "strings"

// specialDockerCompose covers the official Compose v1/v2 filenames, including
// environment-specific variants such as docker-compose.prod.yml.
func isDockerCompose(name string) bool {
	if name == "compose.yml" || name == "compose.yaml" ||
		name == "docker-compose.yml" || name == "docker-compose.yaml" ||
		name == "docker-compose" {
		return true
	}
	for _, suffix := range []string{".yml", ".yaml"} {
		if strings.HasPrefix(name, "docker-compose.") && strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// matchSpecialFile resolves filenames whose identity is more specific than the
// generic language reported by the IDE (e.g. package.json being npm, vite.config.ts being Vite).
// Asset keys match available icon themes, with semantic fallbacks.
func matchSpecialFile(lowerFilename, lowerPath string) (LanguageInfo, bool) {
	// 1. Docker & Container orchestration
	if isDockerCompose(lowerFilename) {
		return LanguageInfo{ID: "docker", Name: "Docker Compose", LargeImage: "docker"}, true
	}
	switch lowerFilename {
	case "dockerfile", "containerfile", ".dockerfile", ".dockerignore", "docker-sync.yml":
		return LanguageInfo{ID: "dockerfile", Name: "Dockerfile", LargeImage: "docker"}, true
	}

	// 2. GitHub Actions & CI/CD workflows
	if strings.Contains(lowerPath, "/.github/workflows/") &&
		(strings.HasSuffix(lowerFilename, ".yml") || strings.HasSuffix(lowerFilename, ".yaml")) {
		return LanguageInfo{ID: "github", Name: "GitHub Actions", LargeImage: "github"}, true
	}
	if strings.Contains(lowerPath, "/.github/") && strings.HasPrefix(lowerFilename, "dependabot.") {
		return LanguageInfo{ID: "dependabot", Name: "Dependabot", LargeImage: "github"}, true
	}

	// 3. JS / TS Package Managers & Runtimes
	switch lowerFilename {
	case "package.json":
		return LanguageInfo{ID: "javascript", Name: "Node Package", LargeImage: "nodejs"}, true
	case "package-lock.json", ".npmrc", ".npmignore":
		return LanguageInfo{ID: "npm", Name: "npm", LargeImage: "npm"}, true
	case "yarn.lock", ".yarnrc", ".yarnrc.yml":
		return LanguageInfo{ID: "yarn", Name: "Yarn", LargeImage: "yarn"}, true
	case "pnpm-lock.yaml", "pnpm-workspace.yaml":
		return LanguageInfo{ID: "pnpm", Name: "pnpm", LargeImage: "pnpm"}, true
	case "bun.lock", "bun.lockb", "bunfig.toml":
		return LanguageInfo{ID: "bun", Name: "Bun", LargeImage: "bun"}, true
	case "deno.json", "deno.jsonc", "deno.lock":
		return LanguageInfo{ID: "deno", Name: "Deno", LargeImage: "deno"}, true
	case ".nvmrc", ".node-version":
		return LanguageInfo{ID: "nodejs", Name: "Node.js", LargeImage: "nodejs"}, true
	}

	// 4. Build tools, dependency manifests & ecosystems
	switch lowerFilename {
	case "build.gradle":
		return LanguageInfo{ID: "groovy", Name: "Gradle", LargeImage: "gradle"}, true
	case "build.gradle.kts":
		return LanguageInfo{ID: "kotlin", Name: "Gradle Kotlin", LargeImage: "kotlin"}, true
	case "settings.gradle", "settings.gradle.kts", "gradle.properties", "gradlew", "gradlew.bat", "libs.versions.toml":
		return LanguageInfo{ID: "gradle", Name: "Gradle", LargeImage: "gradle"}, true
	case "pom.xml":
		return LanguageInfo{ID: "java", Name: "Maven POM", LargeImage: "java"}, true
	case "mvnw", "mvnw.cmd":
		return LanguageInfo{ID: "maven", Name: "Maven", LargeImage: "maven"}, true
	case "cargo.toml", "cargo.lock":
		return LanguageInfo{ID: "rust", Name: "Rust Manifest", LargeImage: "rust"}, true
	case "go.mod", "go.sum", "go.work":
		return LanguageInfo{ID: "go", Name: "Go Modules", LargeImage: "go"}, true
	case "poetry.lock", "poetry.toml":
		return LanguageInfo{ID: "poetry", Name: "Poetry", LargeImage: "poetry"}, true
	case "requirements.txt", "pipfile", "pipfile.lock", "setup.py", "setup.cfg", "pyproject.toml":
		return LanguageInfo{ID: "python", Name: "Python", LargeImage: "python"}, true
	case "environment.yml", "environment.yaml":
		return LanguageInfo{ID: "anaconda", Name: "Conda", LargeImage: "python"}, true
	case "composer.json", "composer.lock":
		return LanguageInfo{ID: "composer", Name: "Composer", LargeImage: "composer"}, true
	case "gemfile", "gemfile.lock":
		return LanguageInfo{ID: "ruby", Name: "Bundler", LargeImage: "ruby"}, true
	case "pubspec.yaml", "pubspec.lock":
		return LanguageInfo{ID: "flutter", Name: "Flutter", LargeImage: "dart"}, true
	case "androidmanifest.xml":
		return LanguageInfo{ID: "android", Name: "Android", LargeImage: "android"}, true
	case "package.swift":
		return LanguageInfo{ID: "swift", Name: "Swift Package", LargeImage: "swift"}, true
	case "podfile", "podfile.lock", "project.pbxproj":
		return LanguageInfo{ID: "xcode", Name: "Xcode", LargeImage: "apple"}, true
	case "project.godot":
		return LanguageInfo{ID: "godot", Name: "Godot", LargeImage: "godot"}, true
	case "chart.yaml", "values.yaml", "helmfile.yaml":
		return LanguageInfo{ID: "helm", Name: "Helm", LargeImage: "docker"}, true
	case "kustomization.yaml", "kustomization.yml":
		return LanguageInfo{ID: "kubernetes", Name: "Kubernetes", LargeImage: "docker"}, true
	case "ansible.cfg", "playbook.yml", "playbook.yaml":
		return LanguageInfo{ID: "ansible", Name: "Ansible", LargeImage: "ansible"}, true
	case "vagrantfile":
		return LanguageInfo{ID: "vagrant", Name: "Vagrant", LargeImage: "vagrant"}, true
	case "jenkinsfile":
		return LanguageInfo{ID: "jenkins", Name: "Jenkins", LargeImage: "groovy"}, true
	case ".gitlab-ci.yml":
		return LanguageInfo{ID: "gitlab", Name: "GitLab CI", LargeImage: "git"}, true
	case ".travis.yml":
		return LanguageInfo{ID: "travis", Name: "Travis CI", LargeImage: "travis"}, true
	case "nginx.conf":
		return LanguageInfo{ID: "nginx", Name: "NGINX", LargeImage: "config"}, true
	case "vercel.json":
		return LanguageInfo{ID: "vercel", Name: "Vercel", LargeImage: "config"}, true
	case "netlify.toml":
		return LanguageInfo{ID: "netlify", Name: "Netlify", LargeImage: "config"}, true
	case "wrangler.toml", "wrangler.json", "wrangler.jsonc":
		return LanguageInfo{ID: "cloudflare", Name: "Cloudflare Workers", LargeImage: "config"}, true
	case "firebase.json", ".firebaserc":
		return LanguageInfo{ID: "firebase", Name: "Firebase", LargeImage: "config"}, true
	case "tauri.conf.json":
		return LanguageInfo{ID: "tauri", Name: "Tauri", LargeImage: "rust"}, true
	case "angular.json":
		return LanguageInfo{ID: "angular", Name: "Angular", LargeImage: "angular"}, true
	case "nx.json":
		return LanguageInfo{ID: "nx", Name: "Nx", LargeImage: "config"}, true
	case "turbo.json":
		return LanguageInfo{ID: "turborepo", Name: "Turborepo", LargeImage: "config"}, true
	case "biome.json", "biome.jsonc":
		return LanguageInfo{ID: "biome", Name: "Biome", LargeImage: "config"}, true
	case "renovate.json", "renovate.json5", ".renovaterc":
		return LanguageInfo{ID: "renovate", Name: "Renovate", LargeImage: "github"}, true
	case ".editorconfig":
		return LanguageInfo{ID: "editorconfig", Name: "EditorConfig", LargeImage: "editorconfig"}, true
	case ".babelrc":
		return LanguageInfo{ID: "babel", Name: "Babel", LargeImage: "javascript"}, true
	}

	// 5. Linters, Formatters & Language configs
	if strings.HasPrefix(lowerFilename, ".eslintrc") || lowerFilename == ".eslintignore" {
		return LanguageInfo{ID: "eslint", Name: "ESLint", LargeImage: "javascript"}, true
	}
	if strings.HasPrefix(lowerFilename, ".prettierrc") || lowerFilename == ".prettierignore" {
		return LanguageInfo{ID: "prettier", Name: "Prettier", LargeImage: "prettier"}, true
	}
	if strings.HasPrefix(lowerFilename, "tsconfig") && strings.HasSuffix(lowerFilename, ".json") {
		return LanguageInfo{ID: "typescript", Name: "TypeScript Config", LargeImage: "typescript"}, true
	}
	if strings.HasPrefix(lowerFilename, "jsconfig") && strings.HasSuffix(lowerFilename, ".json") {
		return LanguageInfo{ID: "javascript", Name: "JavaScript Config", LargeImage: "javascript"}, true
	}
	if strings.HasSuffix(lowerFilename, ".csproj") || strings.HasSuffix(lowerFilename, ".sln") || strings.HasSuffix(lowerFilename, ".slnx") {
		return LanguageInfo{ID: "dotnet", Name: ".NET", LargeImage: "dotnet"}, true
	}

	// 6. Tools by .config. pattern (e.g. vite.config.ts, tailwind.config.js)
	if idx := strings.Index(lowerFilename, ".config."); idx != -1 {
		tool := lowerFilename[:idx]
		switch tool {
		case "vite":
			return LanguageInfo{ID: "vite", Name: "Vite", LargeImage: "nodejs"}, true
		case "vitest":
			return LanguageInfo{ID: "vitest", Name: "Vitest", LargeImage: "nodejs"}, true
		case "tailwind":
			return LanguageInfo{ID: "tailwind", Name: "Tailwind CSS", LargeImage: "css"}, true
		case "postcss":
			return LanguageInfo{ID: "postcss", Name: "PostCSS", LargeImage: "postcss"}, true
		case "next":
			return LanguageInfo{ID: "nextjs", Name: "Next.js", LargeImage: "react"}, true
		case "nuxt":
			return LanguageInfo{ID: "nuxt", Name: "Nuxt", LargeImage: "vue"}, true
		case "svelte":
			return LanguageInfo{ID: "svelte", Name: "Svelte", LargeImage: "svelte"}, true
		case "astro":
			return LanguageInfo{ID: "astro", Name: "Astro", LargeImage: "html"}, true
		case "webpack":
			return LanguageInfo{ID: "webpack", Name: "webpack", LargeImage: "nodejs"}, true
		case "rollup":
			return LanguageInfo{ID: "rollup", Name: "Rollup", LargeImage: "nodejs"}, true
		case "eslint":
			return LanguageInfo{ID: "eslint", Name: "ESLint", LargeImage: "javascript"}, true
		case "prettier":
			return LanguageInfo{ID: "prettier", Name: "Prettier", LargeImage: "prettier"}, true
		case "babel":
			return LanguageInfo{ID: "babel", Name: "Babel", LargeImage: "javascript"}, true
		case "jest":
			return LanguageInfo{ID: "jest", Name: "Jest", LargeImage: "nodejs"}, true
		case "playwright":
			return LanguageInfo{ID: "playwright", Name: "Playwright", LargeImage: "nodejs"}, true
		case "cypress":
			return LanguageInfo{ID: "cypress", Name: "Cypress", LargeImage: "nodejs"}, true
		case "electron-builder", "forge":
			return LanguageInfo{ID: "electron", Name: "Electron", LargeImage: "nodejs"}, true
		case "tsup", "esbuild":
			return LanguageInfo{ID: "esbuild", Name: "esbuild", LargeImage: "nodejs"}, true
		}
	}

	return LanguageInfo{}, false
}
