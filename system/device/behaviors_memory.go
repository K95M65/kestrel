package device

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.autonomous.ai/os/system/domain"
)

// memoryDir is the on-disk inbox for "remember this". Tests override it.
var memoryDir = "/root/local/companion"

var memoryMu sync.Mutex

type memoryLine struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

func memoriesPath() string { return filepath.Join(memoryDir, "memories.jsonl") }
func briefDayPath() string { return filepath.Join(memoryDir, "last_brief_day") }

func AddMemory(text string, maxItems int) (domain.MemoryItem, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return domain.MemoryItem{}, fmt.Errorf("text is required")
	}
	if maxItems <= 0 {
		maxItems = 200
	}
	item := memoryLine{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	memoryMu.Lock()
	defer memoryMu.Unlock()
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		return domain.MemoryItem{}, err
	}
	f, err := os.OpenFile(memoriesPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return domain.MemoryItem{}, err
	}
	enc := json.NewEncoder(f)
	err = enc.Encode(item)
	_ = f.Close()
	if err != nil {
		return domain.MemoryItem{}, err
	}
	_ = trimMemoriesLocked(maxItems)
	return domain.MemoryItem{ID: item.ID, Text: item.Text, CreatedAt: item.CreatedAt}, nil
}

func ListMemories() []domain.MemoryItem {
	memoryMu.Lock()
	defer memoryMu.Unlock()
	return listMemoriesLocked()
}

func CountMemories() int {
	return len(ListMemories())
}

func DeleteMemory(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	memoryMu.Lock()
	defer memoryMu.Unlock()
	items := listMemoriesLocked()
	out := make([]memoryLine, 0, len(items))
	found := false
	for _, it := range items {
		if it.ID == id {
			found = true
			continue
		}
		out = append(out, memoryLine{ID: it.ID, Text: it.Text, CreatedAt: it.CreatedAt})
	}
	if !found {
		return fmt.Errorf("not found")
	}
	return rewriteMemoriesLocked(out)
}

func listMemoriesLocked() []domain.MemoryItem {
	f, err := os.Open(memoriesPath())
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []domain.MemoryItem
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var it memoryLine
		if json.Unmarshal([]byte(line), &it) != nil || it.ID == "" {
			continue
		}
		out = append(out, domain.MemoryItem{ID: it.ID, Text: it.Text, CreatedAt: it.CreatedAt})
	}
	return out
}

func trimMemoriesLocked(max int) error {
	items := listMemoriesLocked()
	if len(items) <= max {
		return nil
	}
	keep := items[len(items)-max:]
	lines := make([]memoryLine, len(keep))
	for i, it := range keep {
		lines[i] = memoryLine{ID: it.ID, Text: it.Text, CreatedAt: it.CreatedAt}
	}
	return rewriteMemoriesLocked(lines)
}

func rewriteMemoriesLocked(items []memoryLine) error {
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		return err
	}
	tmp := memoriesPath() + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, it := range items {
		if err := enc.Encode(it); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, memoriesPath())
}

func saveBriefDay(day string) {
	_ = os.MkdirAll(memoryDir, 0o755)
	_ = os.WriteFile(briefDayPath(), []byte(day+"\n"), 0o644)
}

func loadBriefDay() string {
	b, err := os.ReadFile(briefDayPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
