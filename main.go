package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"
)

type Node struct{
	Part string
	Children map[string]*Node
	Frequency int
	Sequence uint32
	Timestamp time.Time
}

type Trie struct{
	Root *Node
	Sequence uint32 // global sequence
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{Part: "", Children: make(map[string]*Node), Sequence: 1},
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

	cur.Sequence = t.Sequence
	cur.Timestamp = time.Now()
	t.Sequence++
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

	if len(cur.Children) == 0 && len(parts) == len(tokenize(search)) {
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

	if len(node.Children) == 0 {
		results = append(results, strings.TrimSpace(command))
	}

	for _, child := range node.Children {
		results = append(results, getFullCommands(child, command + " " + child.Part)...)
	}

	return results
}

func main() {
	trixie := NewTrixie("./db.trixie")
	curuser, err := user.Current()
	data, err := os.ReadFile(curuser.HomeDir + "/.local/share/hilbish/.hilbish-history")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")

	for _, l := range lines {
		trixie.Trie.Write(l)
	}

	{
		start := time.Now()
		err = trixie.Save()
		if err != nil {
			panic(err)
		}
		duration := time.Since(start)
		fmt.Printf("Save took: %v\n", duration)
	}

	{
		
		startDeserde := time.Now()
		f, err := os.Open("./db.trixie")
		if err != nil {
			panic(err)
		}
		err = trixie.Trie.Deserialize(bufio.NewReader(f))
		if err != nil {
			panic(err)
		}
		f.Close()
		loadDur := time.Since(startDeserde)
		fmt.Printf("Load took: %v\n", loadDur)
	}

	{
		start := time.Now()
		err = trixie.Save()
		if err != nil {
			panic(err)
		}
		duration := time.Since(start)
		fmt.Printf("Save took: %v\n", duration)
	}
}
