/**
 * Natural Language Query Demo
 *
 * Interactive demo of the NL query engine
 * Runs example queries without needing full API server
 *
 * Usage:
 *   export OPENAI_API_KEY="your-api-key"
 *   go run main.go
 */

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"genomevedic/backend/internal/ai"
)

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        GenomeVedic Natural Language Query Demo              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create NL query engine
	engine := ai.NewNLQueryEngine(apiKey)

	// Show example queries
	fmt.Println("📚 Example Queries:")
	examples := engine.GetExamples()
	for i, ex := range examples[:5] {
		fmt.Printf("%d. %s\n", i+1, ex.NaturalLanguage)
	}
	fmt.Println()

	// Interactive mode
	fmt.Println("💬 Enter your queries (or 'quit' to exit):")
	fmt.Println(strings.Repeat("-", 60))

	scanner := bufio.NewScanner(os.Stdin)
	userID := "demo_user"

	for {
		fmt.Print("\nQuery> ")

		if !scanner.Scan() {
			break
		}

		query := strings.TrimSpace(scanner.Text())

		if query == "" {
			continue
		}

		if query == "quit" || query == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Special commands
		if query == "examples" {
			showExamples(engine)
			continue
		}

		if query == "help" {
			showHelp()
			continue
		}

		// Process query
		fmt.Println()
		fmt.Println("🔄 Processing query...")

		result, err := engine.ConvertToSQL(userID, query)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Display result
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("📝 Original Query: %s\n", result.OriginalQuery)
		fmt.Println()

		if result.IsValid {
			fmt.Printf("✅ Generated SQL:\n")
			fmt.Printf("   %s\n", result.GeneratedSQL)
			fmt.Println()

			if result.Explanation != "" {
				fmt.Printf("💡 Explanation: %s\n", result.Explanation)
				fmt.Println()
			}

			fmt.Printf("⏱️  Execution Time: %dms\n", result.ExecutionTimeMs)
		} else {
			fmt.Printf("❌ Validation Failed:\n")
			fmt.Printf("   %s\n", result.ValidationError)
		}
		fmt.Println(strings.Repeat("-", 60))
	}
}

func showExamples(engine *ai.NLQueryEngine) {
	fmt.Println()
	fmt.Println("📚 Available Query Examples:")
	fmt.Println(strings.Repeat("-", 60))

	examples := engine.GetExamples()
	for i, ex := range examples {
		fmt.Printf("\n%d. %s\n", i+1, ex.NaturalLanguage)
		fmt.Printf("   SQL: %s\n", ex.SQL)
		fmt.Printf("   Description: %s\n", ex.Description)
	}
	fmt.Println(strings.Repeat("-", 60))
}

func showHelp() {
	fmt.Println()
	fmt.Println("💡 Available Commands:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("  examples  - Show all example queries")
	fmt.Println("  help      - Show this help message")
	fmt.Println("  quit      - Exit the demo")
	fmt.Println()
	fmt.Println("💡 Example Queries:")
	fmt.Println("  Show me all TP53 mutations")
	fmt.Println("  Find pathogenic variants in BRCA1")
	fmt.Println("  What are variants with MAF > 0.01?")
	fmt.Println("  List all variants on chromosome 17")
	fmt.Println(strings.Repeat("-", 60))
}
