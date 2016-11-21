package rhmap

import (
	"fmt"
	"time"
)

//domain logic that is custom to rhmap

// Service the rhmap domain logic service
type Service struct{}

// UniqueName creates unique names for things
func (s Service) UniqueName(prefix, guid string) string {
	var (
		prefixLen    = 6
		fillerLen    = 4
		timestampLen = 13
		stamp        = fmt.Sprintf("%d", time.Now().UnixNano())
	)
	if len(prefix) < prefixLen {
		prefixLen = len(prefix)
	}
	if len(guid) < fillerLen {
		fillerLen = len(guid)
	}
	return fmt.Sprintf("%s-%s%s", prefix[0:prefixLen], stamp[0:timestampLen], guid[0:fillerLen])
}

// ConsistentName returns the same name for prefix and guid
func (s Service) ConsistentName(prefix, guid string) string {
	return fmt.Sprintf("%s-%s", prefix, guid)
}
