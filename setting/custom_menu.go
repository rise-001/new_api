package setting

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const (
	CustomMenuOptionKey       = "CustomMenuItems"
	CustomMenuLocationChat    = "chat"
	CustomMenuLocationTop     = "top"
	CustomMenuOpenModeEmbed   = "embed"
	CustomMenuOpenModeNewTab  = "new_tab"
	customMenuMaxItems        = 50
	customMenuMaxNameRunes    = 64
	customMenuMaxURLLength    = 2048
)

var customMenuIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type CustomMenuItem struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Name     string `json:"name"`
	Location string `json:"location"`
	OpenMode string `json:"open_mode"`
}

func ValidateCustomMenuItemsJSON(value string) error {
	var items []CustomMenuItem
	if err := common.UnmarshalJsonStr(value, &items); err != nil {
		return fmt.Errorf("custom menus must be a valid JSON array: %w", err)
	}
	if items == nil {
		return fmt.Errorf("custom menus must be a JSON array")
	}
	if len(items) > customMenuMaxItems {
		return fmt.Errorf("custom menus cannot contain more than %d items", customMenuMaxItems)
	}

	seenIDs := make(map[string]struct{}, len(items))
	for index, item := range items {
		itemNumber := index + 1
		if !customMenuIDPattern.MatchString(item.ID) {
			return fmt.Errorf("custom menu item %d has an invalid id", itemNumber)
		}
		if _, exists := seenIDs[item.ID]; exists {
			return fmt.Errorf("custom menu item %d has a duplicate id", itemNumber)
		}
		seenIDs[item.ID] = struct{}{}

		name := strings.TrimSpace(item.Name)
		if name == "" || utf8.RuneCountInString(name) > customMenuMaxNameRunes {
			return fmt.Errorf("custom menu item %d name must contain 1-%d characters", itemNumber, customMenuMaxNameRunes)
		}

		rawURL := strings.TrimSpace(item.URL)
		if rawURL == "" || len(rawURL) > customMenuMaxURLLength {
			return fmt.Errorf("custom menu item %d has an invalid url", itemNumber)
		}
		parsedURL, err := url.ParseRequestURI(rawURL)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return fmt.Errorf("custom menu item %d url must use http or https", itemNumber)
		}

		switch item.Location {
		case CustomMenuLocationChat:
			if item.OpenMode != CustomMenuOpenModeEmbed && item.OpenMode != CustomMenuOpenModeNewTab {
				return fmt.Errorf("custom menu item %d has an invalid open mode", itemNumber)
			}
		case CustomMenuLocationTop:
			if item.OpenMode != CustomMenuOpenModeNewTab {
				return fmt.Errorf("custom menu item %d in top navigation must open in a new tab", itemNumber)
			}
		default:
			return fmt.Errorf("custom menu item %d has an invalid location", itemNumber)
		}
	}

	return nil
}
