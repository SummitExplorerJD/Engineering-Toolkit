package sqliteindex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspaceconfig"
)

const (
	maxCandidateFiles = 20_000
	maxSourceBytes    = 128 << 20
	maxChunks         = 50_000
	maxChunkBytes     = 8 << 10
	maxChunkLines     = 120
	policyVersion     = "engkit.context-policy/v1alpha1"
)

type sourceChunk struct {
	ID            string
	Ordinal       int
	StartLine     int
	EndLine       int
	ContentSHA256 string
	Content       string
}

type sourceRecord struct {
	Path             string
	Kind             string
	Language         string
	SizeBytes        int64
	ModifiedNS       int64
	SHA256           string
	NormalizedSHA256 string
	Chunks           []sourceChunk
}

type sourceInventory struct {
	Sources         []sourceRecord
	Fingerprint     string
	ConfigSHA256    string
	ConfiguredRoots int
	SourceBytes     int64
	ChunkCount      int
	Exclusions      map[string]int
}

var sourceExtensions = map[string]string{
	".c": "c", ".cc": "cpp", ".cpp": "cpp", ".cxx": "cpp", ".cs": "csharp",
	".css": "css", ".dart": "dart", ".go": "go", ".h": "c", ".hh": "cpp",
	".hpp": "cpp", ".hxx": "cpp", ".html": "html", ".java": "java", ".js": "javascript",
	".jsx": "javascript", ".json": "json", ".kt": "kotlin", ".kts": "kotlin",
	".lua": "lua", ".md": "markdown", ".markdown": "markdown", ".php": "php",
	".proto": "protobuf", ".py": "python", ".rb": "ruby", ".rs": "rust",
	".scss": "scss", ".sh": "shell", ".sql": "sql", ".swift": "swift",
	".toml": "toml", ".ts": "typescript", ".tsx": "typescript", ".txt": "text",
	".xml": "xml", ".yaml": "yaml", ".yml": "yaml", ".zig": "zig",
}

var sourceBasenames = map[string]string{
	"CMakeLists.txt": "cmake", "Dockerfile": "dockerfile", "Gemfile": "ruby",
	"Gemfile.lock": "ruby", "Makefile": "make", "Cargo.lock": "toml",
	"Cargo.toml": "toml", "go.mod": "go-module", "go.sum": "go-module",
	"go.work": "go-module", "go.work.sum": "go-module", "package.json": "json",
	"package-lock.json": "json", "pnpm-lock.yaml": "yaml", "pyproject.toml": "toml",
	"requirements.txt": "text", "yarn.lock": "text",
}

var excludedDirectoryNames = map[string]struct{}{
	".git": {}, ".hg": {}, ".svn": {}, ".engkit": {}, ".agents": {}, ".cache": {},
	".next": {}, ".nuxt": {}, ".terraform": {}, ".venv": {}, "deriveddata": {},
	"pods": {}, "__pycache__": {}, "bin": {}, "build": {}, "coverage": {}, "dist": {},
	"node_modules": {}, "obj": {}, "out": {}, "target": {}, "third_party": {},
	"vendor": {}, "venv": {},
}

var errNonText = errors.New("context source is not valid UTF-8 text")

func collectSourceInventory(root string) (sourceInventory, error) {
	config, err := workspaceconfig.Load(root)
	if err != nil {
		return sourceInventory{}, err
	}
	configDigest, err := sourceConfigDigest(config.Context)
	if err != nil {
		return sourceInventory{}, err
	}
	result := sourceInventory{ConfigSHA256: configDigest, Exclusions: make(map[string]int)}
	if config.Context != nil {
		result.ConfiguredRoots = len(config.Context.SourceRoots)
	}
	canonical, err := canonicalCandidates(root)
	if err != nil {
		return sourceInventory{}, err
	}
	seen := make(map[string]sourceRecord, len(canonical))
	for _, relative := range canonical {
		kind := artifactKind(relative)
		item, err := readSourceRecord(root, relative, kind, sourceLanguage(relative), kind == "documentation")
		if err != nil {
			return sourceInventory{}, err
		}
		seen[item.Path] = item
	}
	if config.Context != nil {
		if err := collectConfiguredSourceRoots(root, config.Context, seen, &result); err != nil {
			return sourceInventory{}, err
		}
	}
	paths := make([]string, 0, len(seen))
	folded := make(map[string]string, len(seen))
	for relative := range seen {
		key := strings.ToLower(relative)
		if previous, exists := folded[key]; exists && previous != relative {
			return sourceInventory{}, fmt.Errorf("context source paths collide case-insensitively: %s and %s", previous, relative)
		}
		folded[key] = relative
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		item := seen[relative]
		result.Sources = append(result.Sources, item)
		result.ChunkCount += len(item.Chunks)
		if result.ChunkCount > maxChunks {
			return sourceInventory{}, errors.New("context source inventory exceeds the 50000-chunk limit")
		}
	}
	result.Fingerprint = sourceInventoryFingerprint(result)
	return result, nil
}

func sourceConfigDigest(config *workspaceconfig.Context) (string, error) {
	data, err := json.Marshal(struct {
		Policy  string                   `json:"policy"`
		Context *workspaceconfig.Context `json:"context,omitempty"`
	}{Policy: policyVersion, Context: config})
	if err != nil {
		return "", fmt.Errorf("encode context configuration: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func collectConfiguredSourceRoots(root string, config *workspaceconfig.Context, seen map[string]sourceRecord, result *sourceInventory) error {
	exclusions := append([]string(nil), config.ExcludePaths...)
	sort.Strings(exclusions)
	candidates := 0
	for _, configuredRoot := range config.SourceRoots {
		base := root
		if configuredRoot != "." {
			base = filepath.Join(root, filepath.FromSlash(configuredRoot))
		}
		info, err := os.Lstat(base)
		if err != nil {
			return fmt.Errorf("inspect context source root %s: %w", configuredRoot, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("context source root must be a real directory: %s", configuredRoot)
		}
		if err := rejectLinkedSourceRoot(root, configuredRoot); err != nil {
			return err
		}
		err = filepath.WalkDir(base, func(filename string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relativeOS, err := filepath.Rel(root, filename)
			if err != nil {
				return err
			}
			relative := filepath.ToSlash(relativeOS)
			if relative == "." {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || entry.Type()&os.ModeType != 0 && !entry.IsDir() {
				result.Exclusions["linked_or_non_regular"]++
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if mandatorySourcePath(relative) {
				result.Exclusions["mandatory_path"]++
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if userExcludedSourcePath(relative, exclusions) {
				result.Exclusions["user_excluded"]++
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			candidates++
			if candidates > maxCandidateFiles {
				return errors.New("context source inventory exceeds the 20000-candidate limit")
			}
			if mandatorySecretSourcePath(relative) {
				result.Exclusions["mandatory_secret"]++
				return nil
			}
			language := sourceLanguage(relative)
			if language == "" {
				result.Exclusions["disallowed_extension"]++
				return nil
			}
			if _, exists := seen[relative]; exists {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				result.Exclusions["linked_or_non_regular"]++
				return nil
			}
			if info.Size() > maxIndexedContentBytes {
				result.Exclusions["oversized_file"]++
				return nil
			}
			item, err := readSourceRecord(root, relative, "source", language, true)
			if errors.Is(err, errNonText) {
				result.Exclusions["binary_or_invalid_utf8"]++
				return nil
			}
			if err != nil {
				return err
			}
			if result.SourceBytes+item.SizeBytes > maxSourceBytes {
				return errors.New("context source inventory exceeds the 128 MiB limit")
			}
			result.SourceBytes += item.SizeBytes
			seen[relative] = item
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk context source root %s: %w", configuredRoot, err)
		}
	}
	return nil
}

func rejectLinkedSourceRoot(root, relative string) error {
	if relative == "." {
		return nil
	}
	candidate := root
	for _, segment := range strings.Split(relative, "/") {
		candidate = filepath.Join(candidate, segment)
		info, err := os.Lstat(candidate)
		if err != nil {
			return fmt.Errorf("inspect context source root component %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("context source root must not contain linked components: %s", relative)
		}
	}
	return nil
}

func readSourceRecord(root, relative, kind, language string, content bool) (sourceRecord, error) {
	filename := filepath.Join(root, filepath.FromSlash(relative))
	before, err := os.Lstat(filename)
	if err != nil {
		return sourceRecord{}, fmt.Errorf("inspect context source %s: %w", filename, err)
	}
	if !before.Mode().IsRegular() {
		return sourceRecord{}, fmt.Errorf("context source is not a regular file: %s", filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return sourceRecord{}, fmt.Errorf("open context source %s: %w", filename, err)
	}
	defer file.Close()
	hash := sha256.New()
	var data []byte
	if content && before.Size() <= maxIndexedContentBytes {
		data, err = io.ReadAll(io.TeeReader(file, hash))
	} else {
		_, err = io.Copy(hash, file)
	}
	if err != nil {
		return sourceRecord{}, fmt.Errorf("read context source %s: %w", filename, err)
	}
	after, err := file.Stat()
	if err != nil {
		return sourceRecord{}, fmt.Errorf("reinspect context source %s: %w", filename, err)
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return sourceRecord{}, fmt.Errorf("context source changed while indexing: %s", filename)
	}
	item := sourceRecord{Path: filepath.ToSlash(relative), Kind: kind, Language: language, SizeBytes: before.Size(), ModifiedNS: before.ModTime().UnixNano(), SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil))}
	if data != nil {
		normalized, ok := normalizeSourceText(data)
		if !ok {
			return sourceRecord{}, errNonText
		}
		normalizedSum := sha256.Sum256([]byte(normalized))
		item.NormalizedSHA256 = "sha256:" + hex.EncodeToString(normalizedSum[:])
		item.Chunks = splitSourceChunks(item, normalized)
	}
	return item, nil
}

func normalizeSourceText(data []byte) (string, bool) {
	if len(data) >= 3 && string(data[:3]) == "\xef\xbb\xbf" {
		data = data[3:]
	}
	if !utf8.Valid(data) || strings.IndexByte(string(data), 0) >= 0 {
		return "", false
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n"), true
}

func splitSourceChunks(source sourceRecord, text string) []sourceChunk {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	chunks := make([]sourceChunk, 0, (len(text)/maxChunkBytes)+1)
	var builder strings.Builder
	startLine, endLine, lineCount := 1, 0, 0
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		content := builder.String()
		sum := sha256.Sum256([]byte(content))
		contentDigest := "sha256:" + hex.EncodeToString(sum[:])
		ordinal := len(chunks)
		chunks = append(chunks, sourceChunk{ID: sourceChunkIdentity(source.Path, source.SHA256, ordinal, startLine, endLine, contentDigest), Ordinal: ordinal, StartLine: startLine, EndLine: endLine, ContentSHA256: contentDigest, Content: content})
		builder.Reset()
		lineCount = 0
	}
	for index, line := range lines {
		lineNumber := index + 1
		for len(line) > maxChunkBytes {
			flush()
			cut := maxChunkBytes
			for cut > 0 && !utf8.ValidString(line[:cut]) {
				cut--
			}
			startLine, endLine = lineNumber, lineNumber
			builder.WriteString(line[:cut])
			line = line[cut:]
			flush()
		}
		if builder.Len() > 0 && (builder.Len()+len(line) > maxChunkBytes || lineCount == maxChunkLines) {
			flush()
		}
		if builder.Len() == 0 {
			startLine = lineNumber
		}
		builder.WriteString(line)
		endLine = lineNumber
		lineCount++
	}
	flush()
	return chunks
}

func sourceChunkIdentity(sourcePath, sourceDigest string, ordinal, startLine, endLine int, contentDigest string) string {
	hash := sha256.New()
	for _, value := range []string{indexSchema, sourcePath, sourceDigest, strconv.Itoa(ordinal), strconv.Itoa(startLine), strconv.Itoa(endLine), contentDigest} {
		_, _ = fmt.Fprintf(hash, "%d:%s\x00", len(value), value)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func sourceInventoryFingerprint(value sourceInventory) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, value.ConfigSHA256)
	_, _ = hash.Write([]byte{0})
	for _, item := range value.Sources {
		for _, field := range []string{item.Path, item.Kind, item.Language, item.SHA256, strconv.FormatInt(item.SizeBytes, 10), strconv.FormatInt(item.ModifiedNS, 10)} {
			_, _ = fmt.Fprintf(hash, "%d:%s\x00", len(field), field)
		}
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func mandatorySourcePath(relative string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(relative), "/") {
		if _, excluded := excludedDirectoryNames[strings.ToLower(segment)]; excluded {
			return true
		}
	}
	return false
}

func userExcludedSourcePath(relative string, exclusions []string) bool {
	for _, excluded := range exclusions {
		if relative == excluded || strings.HasPrefix(relative, excluded+"/") {
			return true
		}
	}
	return false
}

func mandatorySecretSourcePath(relative string) bool {
	base := strings.ToLower(path.Base(filepath.ToSlash(relative)))
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	for _, exact := range []string{".netrc", "_netrc", ".npmrc", ".pypirc", "credentials.json", "credentials.yaml", "credentials.yml", "secrets.json", "secrets.yaml", "secrets.yml", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519"} {
		if base == exact {
			return true
		}
	}
	for _, suffix := range []string{".key", ".pem", ".p12", ".pfx", ".jks", ".keystore", ".kdbx", ".ovpn"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

func sourceLanguage(relative string) string {
	base := path.Base(filepath.ToSlash(relative))
	if language := sourceBasenames[base]; language != "" {
		return language
	}
	return sourceExtensions[strings.ToLower(path.Ext(base))]
}

func boundedUTF8Prefix(value string, limit int) (string, bool) {
	if limit <= 0 || len(value) <= limit {
		return value, false
	}
	cut := limit
	for cut > 0 && (value[cut]&0xc0) == 0x80 {
		cut--
	}
	return value[:cut], true
}

func literalFTSQuery(query string) string {
	terms := strings.Fields(query)
	quoted := make([]string, 0, len(terms))
	for _, term := range terms {
		quoted = append(quoted, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, " AND ")
}
