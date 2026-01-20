package common

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/darianmavgo/banquet"
)

// FmtPrintln prints the entire Banquet struct to the console.
func FmtPrintln(bq *banquet.Banquet) {
	b, err := json.MarshalIndent(bq, "", "  ")
	if err != nil {
		log.Printf("Error marshalling Banquet struct: %v", err)
		return
	}
	fmt.Println("----- Banquet Struct -----")
	fmt.Println(string(b))
	fmt.Println("--------------------------")
}

// PrintQuery prints the entire SQL query to the console.
func PrintQuery(query string) {
	fmt.Println("----- SQL Query -----")
	fmt.Println(query)
	fmt.Println("---------------------")
}

// DebugLog prints both the Banquet struct and the SQL query for comparison.
func DebugLog(bq *banquet.Banquet, query string) {
	FmtPrintln(bq)
	PrintQuery(query)
}

// GetBanquetJSON returns the JSON representation of the Banquet struct.
func GetBanquetJSON(bq *banquet.Banquet) string {
	b, err := json.Marshal(bq)
	if err != nil {
		return fmt.Sprintf("Error marshalling: %v", err)
	}
	return string(b)
}
