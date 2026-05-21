package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/tool"
	"gopkg.in/yaml.v3"
)

var skillCreatorNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// SkillCreator Skill 创建工具。
// 目录结构和 SKILL.md 生成规则参考 skilldiscovery CreatorAgent。
type SkillCreator struct {
	skillsDir string
}

// NewSkillCreator 创建 Skill 创建工具。
func NewSkillCreator(skillsDir string) *SkillCreator {
	return &SkillCreator{skillsDir: skillsDir}
}

// Name 返回工具名称。
func (c *SkillCreator) Name() string {
	return "skill_creator"
}

// Description 返回工具描述。
func (c *SkillCreator) Description() string {
	return "按 Open Agent Skills 规范创建本地 Skill。生成包含必需 YAML frontmatter 的 SKILL.md，并可一次性写入 scripts、references、assets 文件；文件内容必须由 Agent 在 files 中提供。不要再用 file_writer 创建 Skill 内文件；不要生成依赖用户配置环境变量、API_KEY占位符或未知密钥的 Skill。"
}

// Version 返回工具版本。
func (c *SkillCreator) Version() string {
	return "1.0.0"
}

// Category 返回工具分类。
func (c *SkillCreator) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回工具参数定义。
func (c *SkillCreator) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"name": {
			Type:        "string",
			Description: "Skill目录名称，必须符合 Open Agent Skills 规范：1-64字符，仅小写字母、数字和连字符；不能以连字符开头或结尾；不能包含连续连字符；必须与frontmatter name一致，例如 code-review-helper",
		},
		"description": {
			Type:        "string",
			Description: "Skill描述，1-1024字符，必须同时描述功能和何时使用，并包含帮助Agent识别任务的关键词",
		},
		"instructions": {
			Type:        "string",
			Description: "SKILL.md正文指令。应遵循渐进式披露：主文件保持精炼，写清分步流程、输入输出示例、边界情况；较长资料放references。若files包含scripts脚本，必须说明何时调用脚本、调用哪个相对路径脚本、传入哪些参数、期望输出是什么。不要要求用户自行配置环境变量、API Key占位符或未知密钥",
		},
		"tags": {
			Type:        "array",
			Description: "可选。Donk本地分类标签，不属于Open Agent Skills核心规范；不会写入SKILL.md frontmatter，但会在创建结果中返回",
		},
		"allowed_tools": {
			Type:        "array",
			Description: "可选。Open Agent Skills 的 allowed-tools 字段，写入frontmatter时会转换为空格分隔字符串；用于预批准工具，例如 Bash(git:*) Read",
		},
		"examples": {
			Type:        "array",
			Description: "Skill使用示例列表，可选",
		},
		"metadata": {
			Type:        "object",
			Description: "可选。Open Agent Skills metadata字段，只接受字符串键值对，用于author、version等附加元数据；不会覆盖内置字段",
		},
		"license": {
			Type:        "string",
			Description: "可选。Open Agent Skills license字段，许可证名称或捆绑许可证文件引用",
		},
		"compatibility": {
			Type:        "string",
			Description: "可选。Open Agent Skills compatibility字段，1-500字符，仅在有环境要求时填写，例如需要git、docker、网络访问等",
		},
		"files": {
			Type:        "array",
			Description: "可选附加文件。每项包含path和content，path只能位于scripts、references或assets目录下，并应在SKILL.md中以相对路径引用。脚本、参考资料、资源文件内容必须一次性放在这里，不要再调用file_writer；包含scripts脚本时，SKILL.md会自动追加脚本使用说明，但instructions仍应写清脚本调用时机和参数；Python脚本优先使用标准库，避免依赖requests等第三方包，除非确有必要并声明script_configs依赖；脚本输出必须使用UTF-8，Python脚本涉及中文输出时应避免依赖系统默认编码，必要时显式reconfigure stdout/stderr 为 utf-8；内容不能依赖用户额外配置环境变量、API Key占位符或未知密钥",
		},
		"script_dependencies": {
			Type:        "object",
			Description: "Donk扩展字段，可选。按语言声明依赖，例如 {\"python\":[\"requests==2.32.3\"],\"javascript\":[\"lodash@4.17.21\"]}。创建包含脚本且脚本需要第三方库时必须填写，运行时会先检查/准备依赖再执行",
		},
		"script_configs": {
			Type:        "object",
			Description: "Donk扩展字段，可选。按脚本文件声明运行配置，key为脚本文件名或相对路径，value可包含language、dependencies、dependency_policy、runtime_version，例如 {\"weather.py\":{\"language\":\"python\",\"dependencies\":[\"requests\"],\"dependency_policy\":\"auto\"}}。创建包含脚本的Skill时建议填写，便于Agent知道运行语言、依赖和执行策略",
		},
		"allow_external_config": {
			Type:        "boolean",
			Description: "是否允许生成依赖外部配置、环境变量、API Key占位符或未知密钥的内容，默认false。除非用户明确要求并提供配置方式，否则不要设置为true",
			Default:     false,
		},
		"overwrite": {
			Type:        "boolean",
			Description: "Skill已存在时是否覆盖，默认false",
			Default:     false,
		},
	}
	schema.Required = []string{"name", "description", "instructions"}
	return schema
}

// Execute 执行 Skill 创建。
func (c *SkillCreator) Execute(ctx *tool.Context) (*tool.Result, error) {
	start := time.Now()
	req, result := c.parseRequest(ctx)
	if result != nil {
		return result, nil
	}

	skillDir := filepath.Join(c.skillsDir, req.Name)
	if err := c.ensureTargetDir(skillDir, req.Overwrite); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	if !req.AllowExternalConfig {
		if err := validateNoExternalConfig(req); err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
		}
	}

	createdDirs, err := c.createSkillDirs(skillDir, req)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}

	skillMD, err := c.createSkillMD(skillDir, req)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	createdFiles, err := c.createAdditionalFiles(skillDir, req.Files)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	loader := skill.NewSkillLoader(c.skillsDir)
	createdSkill, err := loader.LoadByName(req.Name)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("加载创建的Skill失败: %v", err)), nil
	}

	result = tool.NewResult(map[string]any{
		"name":          createdSkill.Name(),
		"description":   createdSkill.Description(),
		"license":       createdSkill.License(),
		"compatibility": createdSkill.Compatibility(),
		"tags":          req.Tags,
		"skill_dir":     skillDir,
		"skill_md":      skillMD,
		"created_dirs":  createdDirs,
		"created_files": createdFiles,
		"overwritten":   req.Overwrite,
		"duration_ms":   time.Since(start).Milliseconds(),
	})
	result.SetExecutionTime(time.Since(start))
	return result, nil
}

type skillCreateRequest struct {
	Name                string
	Description         string
	Instructions        string
	Tags                []string
	AllowedTools        []string
	Examples            []string
	Metadata            map[string]string
	License             string
	Compatibility       string
	Files               []skillCreateFile
	ScriptDependencies  map[string][]string
	ScriptConfigs       map[string]skillCreateScriptConfig
	AllowExternalConfig bool
	Overwrite           bool
}

type skillCreateScriptConfig struct {
	Language         string   `yaml:"language"`
	Dependencies     []string `yaml:"dependencies"`
	DependencyPolicy string   `yaml:"dependency-policy"`
	RuntimeVersion   string   `yaml:"runtime-version"`
}

type skillCreateFile struct {
	Path    string
	Content string
}

func (c *SkillCreator) parseRequest(ctx *tool.Context) (*skillCreateRequest, *tool.Result) {
	name := normalizeSkillCreateName(stringParam(ctx.Params["name"]))
	description := strings.TrimSpace(stringParam(ctx.Params["description"]))
	instructions := strings.TrimSpace(stringParam(ctx.Params["instructions"]))
	if name == "" {
		return nil, tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "name不能为空")
	}
	if !isValidSkillCreateName(name) {
		return nil, tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "name必须符合Open Agent Skills规范：1-64字符，仅小写字母、数字和连字符，不能以连字符开头或结尾，不能包含连续连字符")
	}
	if description == "" {
		return nil, tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "description不能为空")
	}
	if len([]rune(description)) > 1024 {
		return nil, tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "description不能超过1024字符")
	}
	if instructions == "" {
		return nil, tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "instructions不能为空")
	}
	compatibility := strings.TrimSpace(stringParam(ctx.Params["compatibility"]))
	if compatibility != "" && len([]rune(compatibility)) > 500 {
		return nil, tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "compatibility不能超过500字符")
	}

	return &skillCreateRequest{
		Name:                name,
		Description:         description,
		Instructions:        instructions,
		Tags:                normalizeStringList(stringSliceParam(ctx.Params["tags"])),
		AllowedTools:        normalizeStringList(stringSliceParam(ctx.Params["allowed_tools"])),
		Examples:            normalizeStringList(stringSliceParam(ctx.Params["examples"])),
		Metadata:            stringMapParam(ctx.Params["metadata"]),
		License:             strings.TrimSpace(stringParam(ctx.Params["license"])),
		Compatibility:       compatibility,
		Files:               skillCreateFilesParam(ctx.Params["files"]),
		ScriptDependencies:  skillCreateDependenciesParam(ctx.Params["script_dependencies"]),
		ScriptConfigs:       skillCreateScriptConfigsParam(ctx.Params["script_configs"]),
		AllowExternalConfig: boolParam(ctx.Params["allow_external_config"]),
		Overwrite:           boolParam(ctx.Params["overwrite"]),
	}, nil
}

func (c *SkillCreator) ensureTargetDir(skillDir string, overwrite bool) error {
	if c.skillsDir == "" {
		return fmt.Errorf("Skill根目录不能为空")
	}
	if err := os.MkdirAll(c.skillsDir, 0755); err != nil {
		return fmt.Errorf("创建Skill根目录失败: %w", err)
	}

	existingDir, exists, err := c.findExistingSkillDir(filepath.Base(skillDir))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if !overwrite {
		return fmt.Errorf("Skill已存在: %s，如需覆盖请设置overwrite=true", filepath.Base(existingDir))
	}
	if err := os.RemoveAll(existingDir); err != nil {
		return fmt.Errorf("覆盖前删除旧Skill目录失败: %w", err)
	}
	return nil
}

func (c *SkillCreator) findExistingSkillDir(name string) (string, bool, error) {
	entries, err := os.ReadDir(c.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("读取Skill根目录失败: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), name) {
			return filepath.Join(c.skillsDir, entry.Name()), true, nil
		}
	}
	return "", false, nil
}

func (c *SkillCreator) createSkillDirs(skillDir string, req *skillCreateRequest) ([]string, error) {
	dirSet := map[string]bool{skillDir: true}
	for _, file := range req.Files {
		cleanPath, err := cleanSkillCreateFilePath(file.Path)
		if err != nil {
			return nil, err
		}
		dirSet[filepath.Join(skillDir, strings.Split(filepath.ToSlash(cleanPath), "/")[0])] = true
	}
	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败 %s: %w", dir, err)
		}
	}
	return dirs, nil
}

func (c *SkillCreator) createAdditionalFiles(skillDir string, files []skillCreateFile) ([]string, error) {
	created := make([]string, 0, len(files))
	for _, file := range files {
		cleanPath, err := cleanSkillCreateFilePath(file.Path)
		if err != nil {
			return nil, err
		}
		fullPath := filepath.Join(skillDir, cleanPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("创建附加文件目录失败 %s: %w", filepath.Dir(fullPath), err)
		}
		if err := os.WriteFile(fullPath, []byte(file.Content), 0644); err != nil {
			return nil, fmt.Errorf("写入附加文件失败 %s: %w", cleanPath, err)
		}
		created = append(created, fullPath)
	}
	return created, nil
}

func (c *SkillCreator) createSkillMD(skillDir string, req *skillCreateRequest) (string, error) {
	frontmatter := map[string]any{
		"name":        req.Name,
		"description": req.Description,
	}
	if req.License != "" {
		frontmatter["license"] = req.License
	}
	if req.Compatibility != "" {
		frontmatter["compatibility"] = req.Compatibility
	}
	if len(req.Metadata) > 0 {
		frontmatter["metadata"] = req.Metadata
	}
	if len(req.AllowedTools) > 0 {
		frontmatter["allowed-tools"] = strings.Join(req.AllowedTools, " ")
	}
	if len(req.ScriptDependencies) > 0 {
		frontmatter["script-dependencies"] = req.ScriptDependencies
	}
	if len(req.ScriptConfigs) > 0 {
		frontmatter["scripts"] = req.ScriptConfigs
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("YAML序列化失败: %w", err)
	}

	var content strings.Builder
	content.WriteString("---\n")
	content.WriteString(string(yamlData))
	content.WriteString("---\n\n")
	content.WriteString(fmt.Sprintf("# %s\n\n", req.Name))
	content.WriteString(req.Instructions)
	if !strings.HasSuffix(req.Instructions, "\n") {
		content.WriteString("\n")
	}
	content.WriteString(buildExamplesSection(req.Examples))
	content.WriteString(buildFileReferenceSection(req))
	content.WriteString(buildScriptUsageSection(req))

	skillFilePath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFilePath, []byte(content.String()), 0644); err != nil {
		return "", fmt.Errorf("写入SKILL.md失败: %w", err)
	}
	return skillFilePath, nil
}

func buildExamplesSection(examples []string) string {
	if len(examples) == 0 {
		return ""
	}
	var content strings.Builder
	content.WriteString("\n## 输入和输出示例\n\n")
	for i, example := range examples {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, example))
	}
	return content.String()
}

func buildFileReferenceSection(req *skillCreateRequest) string {
	references := req.referenceFiles()
	assets := req.assetFiles()
	if len(references) == 0 && len(assets) == 0 {
		return ""
	}
	var content strings.Builder
	content.WriteString("\n## 附加资源\n\n")
	if len(references) > 0 {
		content.WriteString("按需读取以下参考文件，避免一次性加载不相关上下文：\n")
		for _, path := range references {
			content.WriteString(fmt.Sprintf("- `%s`\n", path))
		}
	}
	if len(assets) > 0 {
		content.WriteString("使用以下静态资源时，按相对路径引用：\n")
		for _, path := range assets {
			content.WriteString(fmt.Sprintf("- `%s`\n", path))
		}
	}
	return content.String()
}

func buildScriptUsageSection(req *skillCreateRequest) string {
	scripts := req.scriptFiles()
	if len(scripts) == 0 {
		return ""
	}

	var content strings.Builder
	content.WriteString("\n## 脚本使用说明\n\n")
	content.WriteString("当执行流程需要运行脚本时，使用 Skill 工具的 `execute_script` 操作调用 `scripts/` 目录中的脚本。Python 脚本会由 Donk script_runner 执行，不依赖系统全局 python；第三方依赖由 python_dependency_manager 管理。\n\n")
	content.WriteString("### 可用脚本\n\n")
	for _, script := range scripts {
		content.WriteString(fmt.Sprintf("- `%s`", script.Path))
		if script.Language != "" {
			content.WriteString(fmt.Sprintf("：语言 `%s`", script.Language))
		}
		if script.Config.DependencyPolicy != "" {
			content.WriteString(fmt.Sprintf("，依赖策略 `%s`", script.Config.DependencyPolicy))
		}
		if len(script.Dependencies) > 0 {
			content.WriteString(fmt.Sprintf("，依赖 `%s`", strings.Join(script.Dependencies, "`, `")))
		}
		content.WriteString("。\n")
	}
	content.WriteString("\n### 编码要求\n\n")
	content.WriteString("- 脚本标准输出和标准错误必须使用 UTF-8 编码，避免中文结果在 Windows 环境中出现乱码。\n")
	content.WriteString("- Python 脚本如需输出中文，可在脚本开头使用 `sys.stdout.reconfigure(encoding=\"utf-8\")` 和 `sys.stderr.reconfigure(encoding=\"utf-8\")`；Donk 运行时也会默认注入 `PYTHONIOENCODING=utf-8` 和 `PYTHONUTF8=1`。\n")
	content.WriteString("\n### 调用方式\n\n")
	if len(scripts) == 1 {
		content.WriteString(fmt.Sprintf("- 该 Skill 只有一个脚本，可调用 `execute_script` 并省略 `params.script`，或显式设置 `params.script` 为 `%s`。\n", scripts[0].Path))
	} else {
		content.WriteString("- 该 Skill 包含多个脚本，调用 `execute_script` 时必须在 `params.script` 中指定要运行的脚本路径。\n")
	}
	content.WriteString("- 如脚本需要参数，在 `params.args` 中按脚本约定传入；执行前应根据用户请求整理参数，执行后解释脚本输出并给出下一步结果。\n")
	return content.String()
}

func (req *skillCreateRequest) scriptFiles() []skillCreateScript {
	scripts := []skillCreateScript{}
	for _, cleanPath := range req.filesByTopDir("scripts") {
		config := req.findScriptConfig(cleanPath)
		language := config.Language
		if language == "" {
			language = inferScriptLanguage(cleanPath)
		}
		dependencies := append([]string{}, config.Dependencies...)
		if len(dependencies) == 0 {
			dependencies = append(dependencies, req.ScriptDependencies[language]...)
		}
		scripts = append(scripts, skillCreateScript{
			Path:         filepath.ToSlash(cleanPath),
			Language:     language,
			Dependencies: normalizeDependencies(dependencies),
			Config:       config,
		})
	}
	sort.Slice(scripts, func(i, j int) bool { return scripts[i].Path < scripts[j].Path })
	return scripts
}

func (req *skillCreateRequest) referenceFiles() []string {
	return req.filesByTopDir("references")
}

func (req *skillCreateRequest) assetFiles() []string {
	return req.filesByTopDir("assets")
}

func (req *skillCreateRequest) filesByTopDir(dir string) []string {
	seen := map[string]bool{}
	paths := []string{}
	for _, file := range req.Files {
		cleanPath, err := cleanSkillCreateFilePath(file.Path)
		if err != nil || !strings.HasPrefix(filepath.ToSlash(cleanPath), dir+"/") {
			continue
		}
		path := filepath.ToSlash(cleanPath)
		if seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

type skillCreateScript struct {
	Path         string
	Language     string
	Dependencies []string
	Config       skillCreateScriptConfig
}

func (req *skillCreateRequest) findScriptConfig(cleanPath string) skillCreateScriptConfig {
	candidates := []string{cleanPath, filepath.ToSlash(cleanPath), filepath.Base(cleanPath), "scripts/" + filepath.Base(cleanPath)}
	for _, candidate := range candidates {
		if config, ok := req.ScriptConfigs[candidate]; ok {
			return config
		}
	}
	return skillCreateScriptConfig{}
}

func inferScriptLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	default:
		return ""
	}
}

func isValidSkillCreateName(name string) bool {
	return len([]rune(name)) >= 1 && len([]rune(name)) <= 64 && skillCreatorNameRegex.MatchString(name) && !strings.HasSuffix(name, "-") && !strings.Contains(name, "--")
}

func normalizeSkillCreateName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

func normalizeStringList(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func validateNoExternalConfig(req *skillCreateRequest) error {
	texts := []string{req.Description, req.Instructions}
	for _, file := range req.Files {
		texts = append(texts, file.Content)
	}
	for _, text := range texts {
		if containsExternalConfigPlaceholder(text) {
			return fmt.Errorf("Skill内容包含外部配置、环境变量、API Key占位符或未知密钥要求；请改为无需用户额外配置的实现，或在用户明确提供配置方式后设置allow_external_config=true")
		}
	}
	return nil
}

func containsExternalConfigPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	patterns := []string{
		"api_key",
		"apikey",
		"your_api_key",
		"your-key",
		"your key",
		"替换为",
		"换成你的",
		"your_",
		"os.getenv",
		"getenv(",
		"process.env",
		"环境变量",
		"env var",
		"需要配置",
		"自行配置",
		"获取api密钥",
		"获取 api 密钥",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func skillCreateDependenciesParam(value interface{}) map[string][]string {
	data, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	result := map[string][]string{}
	for language, raw := range data {
		language = normalizeScriptLanguage(language)
		if language != "python" && language != "javascript" {
			continue
		}
		dependencies := normalizeDependencies(stringSliceParam(raw))
		if len(dependencies) > 0 {
			result[language] = dependencies
		}
	}
	return result
}

func skillCreateScriptConfigsParam(value interface{}) map[string]skillCreateScriptConfig {
	data, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	result := map[string]skillCreateScriptConfig{}
	for scriptName, raw := range data {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		language := normalizeScriptLanguage(stringParam(item["language"]))
		if language != "python" && language != "javascript" {
			language = ""
		}
		policy := strings.ToLower(strings.TrimSpace(stringParam(item["dependency_policy"])))
		if policy == "" && len(stringSliceParam(item["dependencies"])) > 0 {
			policy = "auto"
		}
		if policy != "" && policy != "none" && policy != "auto" {
			policy = "auto"
		}
		config := skillCreateScriptConfig{
			Language:         language,
			Dependencies:     normalizeDependencies(stringSliceParam(item["dependencies"])),
			DependencyPolicy: policy,
			RuntimeVersion:   strings.TrimSpace(stringParam(item["runtime_version"])),
		}
		result[scriptName] = config
	}
	return result
}

func skillCreateFilesParam(value interface{}) []skillCreateFile {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	files := make([]skillCreateFile, 0, len(items))
	for _, item := range items {
		data, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		path := strings.TrimSpace(stringParam(data["path"]))
		content, ok := data["content"].(string)
		if path == "" || !ok {
			continue
		}
		files = append(files, skillCreateFile{Path: path, Content: content})
	}
	return files
}

func cleanSkillCreateFilePath(path string) (string, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "/")
	cleanPath := filepath.Clean(filepath.FromSlash(path))
	if cleanPath == "." || cleanPath == "" || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("非法附加文件路径: %s", path)
	}
	first := strings.Split(filepath.ToSlash(cleanPath), "/")[0]
	if first != "scripts" && first != "references" && first != "assets" {
		return "", fmt.Errorf("附加文件只能写入 scripts、references 或 assets 目录: %s", path)
	}
	if strings.EqualFold(filepath.Base(cleanPath), "SKILL.md") {
		return "", fmt.Errorf("附加文件不能覆盖SKILL.md")
	}
	return cleanPath, nil
}
