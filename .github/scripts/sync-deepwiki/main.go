// Command sync-deepwiki projects the public Symphony DeepWiki into a local GitHub wiki checkout.
package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	sourceRefPattern = `([.\w@+/-]+)(?::(\d+(?:-\d+)?(?:,\s*\d+(?:-\d+)?)*))?`
	repository       = "https://github.com/QuanuX/Symphony"
	deepWiki         = "https://deepwiki.com/QuanuX/Symphony"
	owned            = "<!-- Generated from QuanuX/Symphony DeepWiki. Edit the source documentation for lasting changes. -->"
	beginIndex       = "<!-- BEGIN Symphony DeepWiki index -->"
	endIndex         = "<!-- END Symphony DeepWiki index -->"
	seed             = "Explore the [Symphony documentation on DeepWiki](https://deepwiki.com/QuanuX/Symphony)."
)

var (
	indexRow       = regexp.MustCompile(`^\s*- ([0-9]+(?:\.[0-9]+)*) (\S.*)$`)
	nonSlug        = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	snapshotMarker = regexp.MustCompile(`(?m)^<!-- DeepWiki snapshot ([0-9a-f]{64}) citation-commit ([0-9a-f]{40}) -->$`)
	// Match Markdown links before code spans so a citation inside an existing link stays a single link.
	inlineLink = regexp.MustCompile("(!?)\\[([^\\]\\n]*)\\]\\(([^()\\n]*)\\)(`?)|`(" + sourceRefPattern + ")(`\\(\\)|\\]\\(\\)|\\]`|\\(\\)`|`)")
	citation   = regexp.MustCompile("^" + sourceRefPattern + "$")
	citedFile  = regexp.MustCompile(`([.\w@+/-]+):\d`)
)

type topic struct {
	number, title, slug, body string
}

func parseTopics(index, contents string) ([]topic, error) {
	var topics []topic
	titles := make(map[string]int)
	slugs := make(map[string]bool)
	numbers := make(map[string]bool)
	for _, row := range strings.Split(index, "\n") {
		m := indexRow.FindStringSubmatch(row)
		if m == nil {
			if strings.TrimSpace(row) != "" && !strings.HasPrefix(row, "Available pages for QuanuX/Symphony:") {
				return nil, fmt.Errorf("unexpected DeepWiki index line %q", row)
			}
			continue
		}
		number, title := m[1], strings.TrimSpace(m[2])
		slug := strings.Trim(nonSlug.ReplaceAllString(title, "-"), "-")
		_, duplicateTitle := titles[title]
		if slug == "" || slugs[strings.ToLower(slug)] || duplicateTitle || numbers[number] {
			return nil, fmt.Errorf("duplicate or unsupported DeepWiki topic %q", row)
		}
		titles[title] = len(topics)
		slugs[strings.ToLower(slug)] = true
		numbers[number] = true
		topics = append(topics, topic{number: number, title: title, slug: "DeepWiki-" + slug})
	}
	if len(topics) == 0 || !strings.HasPrefix(contents, "# Page: ") {
		return nil, errors.New("DeepWiki returned an empty or unrecognized export")
	}
	for _, page := range strings.Split(strings.TrimPrefix(contents, "# Page: "), "\n# Page: ") {
		title, body, found := strings.Cut(page, "\n")
		pos, exists := titles[strings.TrimSpace(title)]
		if !found || !exists || topics[pos].body != "" {
			return nil, fmt.Errorf("unexpected or repeated DeepWiki page %q", title)
		}
		// The MCP export separates pages with a horizontal rule.
		body = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(body), "\n---"))
		if body == "" {
			return nil, fmt.Errorf("empty DeepWiki page %q", title)
		}
		topics[pos].body = body
	}
	for _, t := range topics {
		if t.body == "" {
			return nil, fmt.Errorf("missing DeepWiki page %q", t.title)
		}
	}
	return topics, nil
}

func git(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}

type renderer struct {
	commit     string
	paths      map[string]bool
	numbers    map[string]string
	unresolved map[string]bool
	lines      map[string]int
}

// These paths occur in the published wiki. Each replacement is checked against the selected tree.
var sourceAliases = map[string]string{
	"-.github/workflows/release-qualification.yml":                 ".github/workflows/release-qualification.yml",
	".github/beta-readiness/prepare.go":                            "tests/beta-readiness/prepare.go",
	".github/public-beta/execution/ci-repair-197-integration.json": "tests/public-beta/execution/ci-repair-197-integration.json",
	"modules/sbv-engine/M3-READINESS.json":                         "modules/sbv-engine/M2-M3-READINESS.json",
	"tools/qxctl/command_manifest_test.go":                         "tools/qxctl/cmd/qxctl/command_manifest_test.go",
}

func (r *renderer) sourcePath(p string) string {
	p = strings.TrimPrefix(strings.TrimPrefix(p, "/"), "./")
	if !r.paths[p] && r.paths[sourceAliases[p]] {
		return sourceAliases[p]
	}
	return p
}

// Count lines in the pinned Git objects in one call, including the last line if it lacks a newline.
func (r *renderer) countLines(repo, contents string) error {
	files := make(map[string]bool)
	for _, match := range citedFile.FindAllStringSubmatch(contents, -1) {
		p := r.sourcePath(match[1])
		if r.paths[p] {
			files[p] = true
		}
	}
	if len(files) == 0 {
		return nil
	}
	args := []string{"--literal-pathspecs", "grep", "-c", "-I", "-e", "^", r.commit, "--"}
	for p := range files {
		args = append(args, p)
	}
	out, err := git(repo, args...)
	if err != nil {
		var status *exec.ExitError
		if !errors.As(err, &status) || status.ExitCode() != 1 {
			return err
		}
	}
	r.lines = make(map[string]int)
	for _, row := range strings.Split(strings.TrimSpace(out), "\n") {
		if row == "" {
			continue
		}
		item, count, found := strings.Cut(strings.TrimPrefix(row, r.commit+":"), ":")
		n, err := strconv.Atoi(count)
		if !found || err != nil {
			return fmt.Errorf("unexpected Git line-count record %q", row)
		}
		r.lines[item] = n
	}
	return nil
}

func lineAnchor(span string, length int) string {
	first, last, isRange := strings.Cut(span, "-")
	start, err := strconv.Atoi(first)
	if err != nil || start < 1 || start > length {
		return ""
	}
	if !isRange {
		return "#L" + first
	}
	end, err := strconv.Atoi(last)
	if err != nil || end < start {
		return ""
	}
	if end > length {
		end = length
	}
	if end == start {
		return "#L" + first
	}
	return fmt.Sprintf("#L%d-L%d", start, end)
}

func (r *renderer) source(label, target string, isImage bool) string {
	m := citation.FindStringSubmatch(strings.Trim(target, "`"))
	if m == nil {
		return label
	}
	p := r.sourcePath(m[1])
	if !r.paths[p] {
		r.unresolved[p] = true
		return label
	}
	u := url.URL{Path: "/" + p}
	kind := "blob"
	if isImage {
		kind = "raw"
	}
	href := repository + "/" + kind + "/" + r.commit + u.EscapedPath()
	if m[2] == "" {
		return "[" + label + "](" + href + ")"
	}
	ranges := strings.Split(m[2], ",")
	links := make([]string, 0, len(ranges))
	for _, span := range ranges {
		span = strings.TrimSpace(span)
		anchor := lineAnchor(span, r.lines[p])
		if anchor != "" && strings.EqualFold(filepath.Ext(p), ".md") {
			anchor = "?plain=1" + anchor
		}
		text := label
		if len(ranges) > 1 {
			text = m[1] + ":" + span
		}
		links = append(links, "["+text+"]("+href+anchor+")")
	}
	return strings.Join(links, ", ")
}

func (r *renderer) inline(line string) string {
	sourceList := strings.HasPrefix(strings.TrimLeft(line, "* "), "Sources:") || strings.HasPrefix(strings.TrimLeft(line, "* "), "Source:")
	return inlineLink.ReplaceAllStringFunc(line, func(match string) string {
		m := inlineLink.FindStringSubmatch(match)
		if ref := m[5]; ref != "" {
			// The upstream export sometimes misplaces a closing backtick, bracket, or ().
			// Bare backticked words in prose are ordinary inline code.
			if m[7] == "" && ((!sourceList && m[8] == "`") || (!r.paths[ref] && !r.paths[sourceAliases[ref]])) {
				return match
			}
			return r.source("`"+ref+"`", ref, false)
		}
		image, label, target, trailing := m[1], m[2], m[3], m[4]
		if target == "" {
			ref := strings.Trim(label, "`")
			if strings.HasPrefix(label, "`") && !strings.HasSuffix(label, "`") {
				label, trailing = "`"+ref+"`", ""
			}
			return r.source(label, ref, false) + trailing
		}
		if strings.HasPrefix(target, "#") {
			if href, ok := r.numbers[target[1:]]; ok {
				return "[" + label + "](" + href + ")" + trailing
			}
		}
		if citation.MatchString(target) {
			return image + r.source(label, target, image == "!") + trailing
		}
		return match
	})
}

func (r *renderer) page(body string) (string, error) {
	lines := strings.Split(body, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "```") && !strings.HasPrefix(trim, "~~~") {
			out = append(out, r.inline(line))
			continue
		}
		fence := trim[:len(trim)-len(strings.TrimLeft(trim, trim[:1]))]
		language := strings.TrimSpace(strings.TrimPrefix(trim, fence))
		start := i + 1
		for i = start; i < len(lines); i++ {
			close := strings.TrimSpace(lines[i])
			if strings.HasPrefix(close, fence) && strings.Trim(close, fence[:1]) == "" {
				break
			}
		}
		if i == len(lines) {
			return "", errors.New("unclosed code fence in DeepWiki page")
		}
		block := strings.Join(lines[start:i], "\n")
		if language == "mermaid" {
			block = normalizeMermaid(block)
		}
		out = append(out, line, block, lines[i])
	}
	return strings.Join(out, "\n"), nil
}

func indexBlock(text, block string) (string, error) {
	start, end := strings.Index(text, beginIndex), strings.Index(text, endIndex)
	switch {
	case start < 0 && end < 0:
		if strings.TrimSpace(text) == seed || strings.TrimSpace(text) == "" {
			return block + "\n", nil
		}
		return text + "\n\n" + block + "\n", nil
	case start >= 0 && end >= start && strings.Count(text, beginIndex) == 1 && strings.Count(text, endIndex) == 1:
		return text[:start] + block + text[end+len(endIndex):], nil
	default:
		return "", errors.New("wiki index contains incomplete or duplicate managed markers")
	}
}

func readPage(dir, name string) (string, error) {
	p := filepath.Join(dir, name)
	info, err := os.Lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("wiki page %s must be a regular file", name)
	}
	data, err := os.ReadFile(p)
	return string(data), err
}

func publishLocal(dir string, pages map[string]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	stems := make(map[string]string)
	for name := range pages {
		stems[strings.ToLower(strings.TrimSuffix(name, ".md"))] = name
	}
	var stale []string
	for _, entry := range entries {
		name := entry.Name()
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if target := stems[strings.ToLower(stem)]; target != "" && name != target {
			return fmt.Errorf("wiki file %s conflicts with planned page %s", name, target)
		}
		if !strings.HasPrefix(name, "DeepWiki-") || filepath.Ext(name) != ".md" {
			continue
		}
		current, err := readPage(dir, name)
		if err != nil {
			return err
		}
		_, wanted := pages[name]
		managed := strings.HasPrefix(current, owned+"\n")
		if wanted && !managed {
			return fmt.Errorf("refusing to overwrite hand-written wiki page %s", name)
		}
		if !wanted && managed {
			stale = append(stale, name)
		}
	}
	for name, data := range pages {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			return err
		}
	}
	for _, name := range stale {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func run(wiki, repo, input, ref string) error {
	if wiki == "" {
		return errors.New("--wiki-dir is required")
	}
	home, err := readPage(wiki, "Home.md")
	if err != nil {
		return err
	}
	sidebar, err := readPage(wiki, "_Sidebar.md")
	if err != nil {
		return err
	}
	index, err := wikiText("read_wiki_structure", input)
	if err != nil {
		return err
	}
	contents, err := wikiText("read_wiki_contents", input)
	if err != nil {
		return err
	}
	topics, err := parseTopics(index, contents)
	if err != nil {
		return err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(index+"\x00"+contents)))
	if ref == "" {
		ref = "HEAD"
		if m := snapshotMarker.FindStringSubmatch(home); m != nil && m[1] == hash {
			ref = m[2]
		}
	}
	commit, err := git(repo, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return err
	}
	commit = strings.TrimSpace(commit)
	tree, err := git(repo, "ls-tree", "-rz", "--name-only", commit)
	if err != nil {
		return err
	}
	r := renderer{commit: commit, paths: make(map[string]bool), numbers: make(map[string]string), unresolved: make(map[string]bool)}
	for _, p := range strings.Split(strings.TrimSuffix(tree, "\x00"), "\x00") {
		r.paths[p] = true
	}
	if err := r.countLines(repo, contents); err != nil {
		return err
	}
	var nav strings.Builder
	for _, t := range topics {
		href := repository + "/wiki/" + t.slug
		r.numbers[t.number] = href
		indent := strings.Repeat("  ", strings.Count(t.number, "."))
		fmt.Fprintf(&nav, "%s- [%s %s](%s)\n", indent, t.number, t.title, href)
	}
	pages := make(map[string]string)
	for _, t := range topics {
		body, err := r.page(t.body)
		if err != nil {
			return fmt.Errorf("%s: %w", t.title, err)
		}
		pages[t.slug+".md"] = owned + "\n\n" + body + "\n"
	}
	marker := fmt.Sprintf("<!-- DeepWiki snapshot %s citation-commit %s -->", hash, commit)
	homeBlock := beginIndex + "\n" + marker + "\n## Symphony documentation\n\nExplore Symphony’s architecture, tools, and release process. These pages are synchronized from [DeepWiki](" + deepWiki + ").\n\n" + nav.String() + endIndex
	pages["Home.md"], err = indexBlock(home, homeBlock)
	if err != nil {
		return err
	}
	sidebarBlock := beginIndex + "\n[Symphony Wiki home](" + repository + "/wiki/Home)\n\n" + nav.String() + endIndex
	pages["_Sidebar.md"], err = indexBlock(sidebar, sidebarBlock)
	if err != nil {
		return err
	}
	if err := publishLocal(wiki, pages); err != nil {
		return err
	}
	var missing []string
	for p := range r.unresolved {
		missing = append(missing, p)
	}
	sort.Strings(missing)
	for _, p := range missing {
		fmt.Fprintf(os.Stderr, "No matching source file; kept citation as text: %s\n", p)
	}
	fmt.Printf("Rendered %d DeepWiki topics and navigation. Citation commit %s.\n", len(topics), commit)
	return nil
}

func main() {
	wiki := flag.String("wiki-dir", "", "local GitHub wiki checkout")
	repo := flag.String("repo-dir", ".", "Symphony Git checkout")
	input := flag.String("input-dir", "", "read saved structure.sse and contents.sse responses instead of contacting DeepWiki")
	ref := flag.String("source-ref", "", "override the citation revision (defaults to the previous revision for an unchanged snapshot, otherwise HEAD)")
	flag.Parse()
	if err := run(*wiki, *repo, *input, *ref); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
