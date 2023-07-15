package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type TrieNode struct {
	children    map[rune]*TrieNode
	isEndOfWord bool
	frequency   int
	prefix      string
}

type AutocompleteSystem struct {
	root *TrieNode
}

func NewTrieNode() *TrieNode {
	return &TrieNode{
		children:    make(map[rune]*TrieNode),
		isEndOfWord: false,
		frequency:   0,
		prefix:      "",
	}
}

func NewAutocompleteSystem() *AutocompleteSystem {
	return &AutocompleteSystem{
		root: NewTrieNode(),
	}
}

func (a *AutocompleteSystem) Insert(word string, frequency int) {
	current := a.root
	prefix := ""

	for _, ch := range word {
		prefix += string(ch)

		node, found := current.children[ch]

		if !found {
			node = NewTrieNode()
			node.prefix = prefix
			current.children[ch] = node
		}

		current = node
	}

	current.isEndOfWord = true
	current.frequency = frequency
}

func (a *AutocompleteSystem) Search(prefix string) []*TrieNode {
	result := make([]*TrieNode, 0)
	current := a.root

	for _, ch := range prefix {
		node, found := current.children[ch]
		if !found {
			return result
		}

		current = node
	}

	a.dfs(current, &result)
	return result
}

func (a *AutocompleteSystem) dfs(node *TrieNode, result *[]*TrieNode) {
	if node.isEndOfWord {
		*result = append(*result, node)
	}

	for _, child := range node.children {
		a.dfs(child, result)
	}
}

func AutocompleteHandler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")

	if prefix == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Menerapkan logika autocomplete
	results := autocomplete.Search(strings.ToLower(prefix))

	// Mengurutkan hasil berdasarkan frekuensi secara descending
	sortResults(results)

	// Mengambil hanya prefix dari kata-kata yang cocok
	suggestions := make([]string, len(results))
	for i, node := range results {
		suggestions[i] = node.prefix
	}

	// Mengembalikan hasil dalam format JSON
	response := map[string]interface{}{
		"suggestions": suggestions,
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func sortResults(results []*TrieNode) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].frequency > results[j].frequency
	})
}

var autocomplete *AutocompleteSystem

func PrintTrie(node *TrieNode, level int) {
	if node == nil {
		return
	}

	// Cetak prefix saat ini dengan indentation sesuai tingkat
	fmt.Printf("%s%s\n", strings.Repeat("\t", level), node.prefix)

	// Cetak setiap child node secara rekursif
	children := make([]*TrieNode, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, child)
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].prefix < children[j].prefix
	})

	for _, child := range children {
		PrintTrie(child, level+1)
	}
}

func main() {
	autocomplete = NewAutocompleteSystem()

	//Buka file CSV keywords sebagai data awal untuk mengisi Trie
	file, err := os.Open("keywords.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	lines, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for i := 1; i < len(lines); i++ {
		row := lines[i]
		if len(row) >= 2 {
			kataKunci := row[0]
			frekuensi, err := strconv.Atoi(row[1])
			if err != nil {
				log.Printf("Error parsing frekuensi: %v", err)
				continue
			}

			// Masukkan keywords ke Trie dengan frekuensi
			autocomplete.Insert(kataKunci, frekuensi)
		}
	}

	http.HandleFunc("/autocomplete", AutocompleteHandler)
	http.HandleFunc("/search", SearchHandler)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))

	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
