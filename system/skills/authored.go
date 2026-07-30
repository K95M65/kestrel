package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// User-authored skills — the web UI's "Write skill" form (chat composer
// "+" → Skills → Write skill) submits a name/description/instructions triple
// that lands on disk as <skillsDir>/<name>/SKILL.md.
//
// The rendering + write live here, not in a runtime package, because only the
// TARGET DIRECTORY differs per agentic runtime — the same reason the skill
// watchers in runtimes/openclaw and runtimes/hermes are near-copies that differ
// only in their skillsDir. Each runtime's AgentGateway.SaveSkill passes its own
// dir; runtimes with no device-writable skills dir don't call this at all.

// ErrInvalidSkillName is returned when a skill name is empty or has an unsafe
// shape. The name becomes a directory under the runtime's skills dir and is how
// the agent addresses the skill, so it is restricted to the same slug shape the
// rest of the skill tooling uses.
var ErrInvalidSkillName = errors.New("invalid skill name")

// ErrSkillExists is returned when a skill directory of that name is already
// present. Authoring never silently overwrites an existing skill — an OTA- or
// store-installed skill of the same name would be destroyed.
var ErrSkillExists = errors.New("skill already exists")

// skillNamePattern allows only lowercase letters, digits, dash and underscore
// (mirrors roleNamePattern in runtimes/openclaw/role_skills.go).
var skillNamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// maxSkillNameLen bounds the directory name. Long enough for descriptive slugs
// like "weekly-status-report", short enough to stay well inside any filesystem
// limit once nested under the runtime's skills dir.
const maxSkillNameLen = 64

// ValidateSkillName reports whether name is usable as a skill directory.
func ValidateSkillName(name string) error {
	if len(name) > maxSkillNameLen {
		return fmt.Errorf("%w: longer than %d characters", ErrInvalidSkillName, maxSkillNameLen)
	}
	if !skillNamePattern.MatchString(name) {
		return fmt.Errorf("%w: %q (lowercase letters, digits, _ and - only)", ErrInvalidSkillName, name)
	}
	return nil
}

// RenderSkillMarkdown builds the SKILL.md body: YAML front-matter carrying
// name + description, then the instructions as the markdown body. Matches the
// shape of the skills shipped in skills/ — the agent reads the front-matter to
// decide whether to load the skill, and the body once it does.
//
// The description is flattened to a single line: a newline inside an unquoted
// YAML scalar would terminate the value and corrupt the front-matter block.
func RenderSkillMarkdown(name, description, instructions string) string {
	desc := strings.Join(strings.Fields(description), " ")

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", name)
	fmt.Fprintf(&b, "description: %s\n", desc)
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimRight(instructions, "\n"))
	b.WriteString("\n")
	return b.String()
}

// ErrSkillNotFound is returned when a skill directory isn't there to remove.
var ErrSkillNotFound = errors.New("skill not found")

// DeleteSkill removes <skillsDir>/<name> and everything under it, returning the
// path it deleted. Name is validated first, so a caller can never be tricked into
// deleting outside the skills dir.
//
// Not idempotent on purpose: a missing skill returns ErrSkillNotFound rather than
// success, so a stale UI or a double-send is visible to the caller instead of
// silently reported as a deletion that never happened.
//
// The runtime is NOT restarted; every backend with a skills dir re-reads it per
// session, the same contract the write paths rely on.
func DeleteSkill(skillsDir, name string) (string, error) {
	if err := ValidateSkillName(name); err != nil {
		return "", err
	}
	if skillsDir == "" {
		return "", errors.New("skills dir is not configured")
	}

	dir := filepath.Join(skillsDir, name)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		// Something is at that path but it isn't a skill — refuse rather than
		// delete a file the caller didn't mean to name.
		return "", fmt.Errorf("%w: %s is not a skill directory", ErrSkillNotFound, name)
	}

	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("remove %s: %w", dir, err)
	}
	return dir, nil
}

// DeleteSkillFrom removes a skill from the first root that has it, for runtimes
// that namespace their skills dir (Hermes). Roots are tried in the order given —
// pass the device-owned root first, matching ListInstalledFrom's precedence, so
// an uninstall hits the same skill the listing showed.
func DeleteSkillFrom(name string, dirs ...string) (string, error) {
	var lastErr error
	for _, dir := range dirs {
		path, err := DeleteSkill(dir, name)
		if err == nil {
			return path, nil
		}
		// A name/config problem is fatal for every root — only keep looking when
		// this particular root simply didn't have the skill.
		if !errors.Is(err, ErrSkillNotFound) {
			return "", err
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}
	return "", lastErr
}

// WriteAuthoredSkill creates <skillsDir>/<name>/SKILL.md from an authored
// draft and returns the path it wrote. Refuses to clobber an existing skill
// directory (ErrSkillExists) so a store- or OTA-installed skill can never be
// overwritten by an authoring mistake.
//
// The runtime is NOT restarted: every backend that has a skills dir picks new
// files up per session (openclaw via skills.load.watch), which is the same
// contract InstallRoleSkills relies on.
func WriteAuthoredSkill(skillsDir, name, description, instructions string) (string, error) {
	if err := ValidateSkillName(name); err != nil {
		return "", err
	}
	if strings.TrimSpace(description) == "" {
		return "", errors.New("description is required")
	}
	if strings.TrimSpace(instructions) == "" {
		return "", errors.New("instructions are required")
	}
	if skillsDir == "" {
		return "", errors.New("skills dir is not configured")
	}

	dir := filepath.Join(skillsDir, name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("%w: %s", ErrSkillExists, name)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat %s: %w", dir, err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create skill dir %s: %w", dir, err)
	}

	path := filepath.Join(dir, "SKILL.md")
	content := RenderSkillMarkdown(name, description, instructions)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		// Don't leave a half-made skill dir behind for the agent to load.
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}
