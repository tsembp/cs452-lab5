/*
Package main implements the Index Partitioner tool.

OVERVIEW:
This tool demonstrates "Document Partitioning" (also known as Local Indexing),
the architectural pattern used by the original Google search engine to scale
horizontal search.

Instead of putting all documents for a single word onto one server (Term Partitioning),
we distribute the documents themselves randomly across N independent shards.
This means:
  1. Every shard holds a subset of the global document corpus.
  2. A single word (e.g., "apple") will appear in multiple shards, but each
     shard's index entry for "apple" will only reference the documents assigned
     to that local shard.
  3. To search for a keyword, a coordinator must broadcast the query to ALL
     shards in parallel, then merge the document lists returned by each shard.

HOW THIS TOOL WORKS:
Given a unified global index file containing lines formatted as:
    <keyword>: <count> <doc1>, <doc2>, <doc3>...

This partitioner processes the file line-by-line, parsing the document list.
For each document, it rolls a pseudo-random die (or uses a seed) to assign it to
one of the N partitions. It then writes out N smaller, independent partition files.

USAGE:
  go run tools/partitioner/main.go \
    -input test/data/global_index.txt \
    -n 3 \
    -output "test/data/shard-%d.txt" \
    -seed 42
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
)

// Rand defines the interface for randomness (for testability)
type Rand interface {
	Intn(n int) int
}

func main() {
	var inputFile string
	var numPartitions int
	var outputPattern string
	var seed int64	

	flag.StringVar(&inputFile, "input", "", "Path to the input index file")
	flag.IntVar(&numPartitions, "n", 2, "Number of partitions to create")
	flag.StringVar(&outputPattern, "output", "", "Pattern for output file names (must include '%d'), e.g., 'out/index-part-%d.txt'")
	flag.Int64Var(&seed, "seed", 42, "Seed for deterministic partitioning")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <file> -n <partitions> -output <pattern>\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintf(os.Stderr, "  go run main.go -input data.txt -n 3 -output \"out/part-%%d.txt\"\n")
	}
	flag.Parse()

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Missing required flag: -input")
		flag.Usage()
		os.Exit(1)
	}
	if numPartitions < 1 {
		fmt.Fprintln(os.Stderr, "Number of partitions must be >= 1")
		flag.Usage()
		os.Exit(1)
	}
	if outputPattern == "" {
		fmt.Fprintln(os.Stderr, "Missing required flag: -output")
		flag.Usage()
		os.Exit(1)
	}
	if !strings.Contains(outputPattern, "%d") {
		fmt.Fprintf(os.Stderr, "-output pattern must include '%%d' to indicate partition number position")
		flag.Usage()
		os.Exit(1)
	}

	// Use seeded randomness
	r := rand.New(rand.NewSource(seed))

	if err := PartitionIndexFile(inputFile, numPartitions, outputPattern, r); err != nil {
		log.Fatalf("Partitioning failed: %v", err)
	}

	fmt.Printf("Partitioning complete: %d partitions written using pattern '%s'\n", numPartitions, outputPattern)
}

// PartitionIndexFile partitions the input index file into numPartitions files using outputPattern.
// The outputPattern must include '%d' for partition index insertion.
// Randomness for partitioning is controlled by r.
func PartitionIndexFile(inputFile string, numPartitions int, outputPattern string, r Rand) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// Prepare output writers and files
	writers := make([]*bufio.Writer, numPartitions)
	files := make([]*os.File, numPartitions)
	for i := 0; i < numPartitions; i++ {
		outFile := fmt.Sprintf(outputPattern, i)
		f, err := os.Create(outFile)
		if err != nil {
			// Close any opened files so far
			for j := 0; j < i; j++ {
				writers[j].Flush()
				files[j].Close()
			}
			return fmt.Errorf("failed to create output file %q: %w", outFile, err)
		}
		files[i] = f
		writers[i] = bufio.NewWriter(f)
	}

	scanner := bufio.NewScanner(file)

	// Allocate a custom buffer to support lines up to 16 MB
    const maxCapacity = 16 * 1024 * 1024
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		keyword := strings.TrimSpace(parts[0])

		rest := strings.TrimSpace(parts[1])
		colonParts := strings.SplitN(rest, " ", 2)
		if len(colonParts) != 2 {
			continue
		}
		docList := strings.Split(strings.TrimSpace(colonParts[1]), ",")

		docsPerPartition := make([][]string, numPartitions)
		for _, doc := range docList {
			doc = strings.TrimSpace(doc)
			if doc == "" {
				continue
			}
			p := r.Intn(numPartitions)
			docsPerPartition[p] = append(docsPerPartition[p], doc)
		}

		for i := 0; i < numPartitions; i++ {
			if len(docsPerPartition[i]) == 0 {
				continue
			}
			outLine := fmt.Sprintf("%s: %d %s\n", keyword, len(docsPerPartition[i]), strings.Join(docsPerPartition[i], ", "))
			if _, err := writers[i].WriteString(outLine); err != nil {
				return fmt.Errorf("failed to write to partition %d: %w", i, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	for i := 0; i < numPartitions; i++ {
		if err := writers[i].Flush(); err != nil {
			return fmt.Errorf("error flushing writer for partition %d: %w", i, err)
		}
		if err := files[i].Close(); err != nil {
			return fmt.Errorf("error closing file for partition %d: %w", i, err)
		}
	}

	return nil
}
