package device

import (
	"fmt"
	"strings"

	"go.autonomous.ai/os/system/server/config"
)

// ApplyRotatedLLMAPIKey writes newKey into config.json when the device is
// still using the previous Grok access token (or has none). If the operator
// has switched brains, the file is left alone.
func (s *Service) ApplyRotatedLLMAPIKey(oldKey, newKey string) error {
	newKey = strings.TrimSpace(newKey)
	if newKey == "" {
		return fmt.Errorf("empty access token")
	}
	oldKey = strings.TrimSpace(oldKey)
	var apply bool
	if err := s.config.WithLockSave(func(c *config.Config) {
		if !isXAIBase(c.LLMBaseURL) {
			return
		}
		if c.LLMAPIKey != "" && oldKey != "" && c.LLMAPIKey != oldKey {
			return
		}
		if c.LLMAPIKey == newKey {
			return
		}
		c.LLMAPIKey = newKey
		apply = true
	}); err != nil {
		return err
	}
	if apply {
		s.syncLLMToGateway(updateChanges{apiKey: true})
	}
	return nil
}

func isXAIBase(u string) bool {
	return strings.Contains(strings.ToLower(u), "api.x.ai")
}
