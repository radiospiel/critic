// Package fnmatch provides shell-style pattern matching (fnmatch) functionality.
// Patterns are converted to regular expressions and cached using an LRU cache.
package fnmatch

import (
	"errors"
	"regexp"
	"strings"
	"sync"
)

// Fnmatcher represents a compiled fnmatch pattern.
type Fnmatcher struct {
	pattern string
	re      *regexp.Regexp
}

// MustCompile compiles an fnmatch pattern and returns a Fnmatcher.
// It panics if the pattern is invalid.
func MustCompile(pattern string) *Fnmatcher {
	regexStr, err := fnmatchToRegex(pattern)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(regexStr)
	return &Fnmatcher{
		pattern: pattern,
		re:      re,
	}
}

// Match checks if the path matches the fnmatch pattern.
func (f *Fnmatcher) Match(path string) bool {
	return f.re.MatchString(path)
}

// lruCache is a simple LRU cache for compiled fnmatch patterns.
type lruCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*lruNode
	head     *lruNode // most recently used
	tail     *lruNode // least recently used
}

type lruNode struct {
	key     string
	matcher *Fnmatcher
	prev    *lruNode
	next    *lruNode
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*lruNode),
	}
}

func (c *lruCache) get(pattern string) (*Fnmatcher, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.items[pattern]
	if !ok {
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(node)
	return node.matcher, true
}

func (c *lruCache) put(pattern string, matcher *Fnmatcher) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.items[pattern]; ok {
		// Already exists, move to front
		c.moveToFront(node)
		return
	}

	// Create new node
	node := &lruNode{
		key:     pattern,
		matcher: matcher,
	}

	// Add to front
	c.addToFront(node)
	c.items[pattern] = node

	// Evict if over capacity
	if len(c.items) > c.capacity {
		c.evictLRU()
	}
}

func (c *lruCache) moveToFront(node *lruNode) {
	if node == c.head {
		return
	}

	// Remove from current position
	c.removeNode(node)

	// Add to front
	c.addToFront(node)
}

func (c *lruCache) removeNode(node *lruNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
}

func (c *lruCache) addToFront(node *lruNode) {
	node.prev = nil
	node.next = c.head

	if c.head != nil {
		c.head.prev = node
	}
	c.head = node

	if c.tail == nil {
		c.tail = node
	}
}

func (c *lruCache) evictLRU() {
	if c.tail == nil {
		return
	}

	// Remove from map
	delete(c.items, c.tail.key)

	// Remove tail node
	c.removeNode(c.tail)
}

// Global cache with capacity of 256
var globalCache = newLRUCache(256)

// Fnmatch checks if the path matches the fnmatch pattern.
// Compiled patterns are cached using an LRU cache with a capacity of 256.
func Fnmatch(pattern, path string) bool {
	matcher, ok := globalCache.get(pattern)
	if !ok {
		matcher = MustCompile(pattern)
		globalCache.put(pattern, matcher)
	}
	return matcher.Match(path)
}

// fnmatchToRegex converts an fnmatch pattern to a regular expression string.
func fnmatchToRegex(pattern string) (string, error) {
	var buf strings.Builder
	buf.WriteString("^")

	i := 0
	for i < len(pattern) {
		c := pattern[i]
		switch c {
		case '*':
			buf.WriteString(".*")
		case '?':
			buf.WriteByte('.')
		case '[':
			// Find closing bracket
			j := i + 1
			// Handle [!...] and []...] edge cases
			if j < len(pattern) && (pattern[j] == '!' || pattern[j] == '^') {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				return "", errors.New("unclosed bracket")
			}
			// Copy bracket expression, converting ! to ^
			buf.WriteByte('[')
			if i+1 < len(pattern) && pattern[i+1] == '!' {
				buf.WriteByte('^')
				buf.WriteString(pattern[i+2 : j])
			} else {
				buf.WriteString(pattern[i+1 : j])
			}
			buf.WriteByte(']')
			i = j
		case '\\':
			// Escape next character
			if i+1 < len(pattern) {
				i++
				buf.WriteString(regexp.QuoteMeta(string(pattern[i])))
			}
		default:
			// Escape regex metacharacters
			buf.WriteString(regexp.QuoteMeta(string(c)))
		}
		i++
	}

	buf.WriteByte('$')
	return buf.String(), nil
}
