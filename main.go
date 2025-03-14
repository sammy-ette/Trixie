package main

import (
	"fmt"
	"strings"
)

type Node struct{
	Part string
	Children map[string]*Node
	Terminal bool
	Frequency int
}

type Trie struct{
	Root *Node
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{Children: make(map[string]*Node)},
	}
}

func (t *Trie) Write(command string) {
	parts := tokenize(command)
	cur := t.Root

	for _, part := range parts {
		if cur.Children == nil {
			cur.Children = make(map[string]*Node)
		}

		if _, ok := cur.Children[part]; !ok {
			cur.Children[part] = &Node{Part: part}
		}

		cur = cur.Children[part]
		cur.Frequency++
	}

	cur.Terminal = true
}

// Query returns a slice of commands that match the query. It is prefix-based and
// cannot do substring searching.
func (t *Trie) Query(search string) []string {
	parts := tokenize(search)
	cur := t.Root

	for _, part := range parts {
		if cur.Children == nil {
			return []string{}
		}

		if next, ok := cur.Children[part]; ok {
			cur = next
		} else {
			// TODO: substring check?
			// like, the check for 'git com' should return the git commit stored commands.
			return []string{}
		}
	}

	if cur.Terminal && len(parts) == len(tokenize(search)) {
		return []string{search}
	}

	return getFullCommands(cur, strings.Join(parts, " "))
}

func (t *Trie) AllCommands() []string {
	return getFullCommands(t.Root, "")
}

func tokenize(command string) []string {
	return strings.Fields(command)
}

func getFullCommands(node *Node, command string) []string {
	results := []string{}

	if node.Terminal {
		results = append(results, strings.TrimSpace(command))
	}

	for _, child := range node.Children {
		results = append(results, getFullCommands(child, command + " " + child.Part)...)
	}

	return results
}

func main() {
	trie := NewTrie()
	trie.Write("ls -l")
	trie.Write("ls -a")
	trie.Write("git commit -m 'lol'")
	trie.Write("git commit -m 'lmao'")
	trie.Write("git commit -am 'lmao'")
	trie.Write("rm -rf / --no-preserve-root")

	fmt.Println("Autocomplete 'ls':", trie.Query("ls"))
	fmt.Println("Autocomplete 'git':", trie.Query("git"))
	fmt.Println("Autocomplete 'git com':", trie.Query("git com"))
	fmt.Println("All Commands:", trie.AllCommands())
}
