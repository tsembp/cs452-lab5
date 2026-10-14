package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
)

const (
	wordlistURL  = "https://raw.githubusercontent.com/dwyl/english-words/master/words_alpha.txt"
	localCacheFile = "words_alpha.txt"
)

// ensureWordlist checks if the 370k+ wordlist is cached locally. If not, it downloads it.
func ensureWordlist() ([]string, error) {
	if _, err := os.Stat(localCacheFile); os.IsNotExist(err) {
		fmt.Printf("Local wordlist %q not found. Downloading ~4MB English dictionary...\n", localCacheFile)
		resp, err := http.Get(wordlistURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download wordlist: %w", err)
		}
		defer resp.Body.Close()

		out, err := os.Create(localCacheFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache file: %w", err)
		}
		defer out.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			return nil, fmt.Errorf("failed to write wordlist to disk: %w", err)
		}
		fmt.Println("Wordlist cached successfully.")
	}

	file, err := os.Open(localCacheFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			words = append(words, word)
		}
	}
	return words, scanner.Err()
}

func formatFilename(id int) string {
	return fmt.Sprintf("pg-%06d.txt", id)
}

func main() {
	outputFile := flag.String("output", "global_index.txt", "Path to save the generated global index file")
	numWords := flag.Int("num_words", 100000, "Number of unique real words to select (supports up to 370,000)")
	numDocs := flag.Int("num_docs", 5000, "Total number of simulated pg-000000.txt files")
	seed := flag.Int64("seed", 42, "Seed for reproducible index generation")
	flag.Parse()

	// 1. Fetch/Load the 370k+ English word pool
	wordPool, err := ensureWordlist()
	if err != nil {
		fmt.Printf("Error loading wordlist: %v\n", err)
		os.Exit(1)
	}

	if *numWords > len(wordPool) {
		fmt.Printf("Warning: capping requested words to pool maximum of %d\n", len(wordPool))
		*numWords = len(wordPool)
	}

	// 2. Select a deterministic subset of real words
	r := rand.New(rand.NewSource(*seed))
	permsWords := r.Perm(len(wordPool))
	var selectedWords []string
	for i := 0; i < *numWords; i++ {
		selectedWords = append(selectedWords, wordPool[permsWords[i]])
	}
	sort.Strings(selectedWords) // Sort alphabetical for standard indexing structure

	// 3. Generate Zipfian document-to-word mappings
	zipfDocCount := rand.NewZipf(r, 1.15, 2.0, uint64(*numDocs-1))

	file, err := os.Create(*outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Printf("Generating %d real words over %d simulated files into %s...\n", *numWords, *numDocs, *outputFile)

	for _, word := range selectedWords {
		count := int(zipfDocCount.Uint64()) + 1
		if count > *numDocs {
			count = *numDocs
		}

		permsDocs := r.Perm(*numDocs)
		var docIDs []int
		for i := 0; i < count; i++ {
			docIDs = append(docIDs, permsDocs[i]+1)
		}
		sort.Ints(docIDs)

		var docNames []string
		for _, id := range docIDs {
			docNames = append(docNames, formatFilename(id))
		}

		line := fmt.Sprintf("%s: %d %s\n", word, len(docNames), strings.Join(docNames, ", "))
		if _, err := file.WriteString(line); err != nil {
			fmt.Printf("Error writing to index: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Global index generation complete.")
}