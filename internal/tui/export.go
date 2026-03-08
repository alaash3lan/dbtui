package tui

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
)

// exportCSV writes columns and rows as standard CSV to the given path.
func exportCSV(columns []string, rows [][]string, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(columns); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// exportJSON writes columns and rows as a JSON array of objects to the given path.
// Each row becomes a map of column name to cell value.
func exportJSON(columns []string, rows [][]string, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		record := make(map[string]string, len(columns))
		for i, col := range columns {
			if i < len(row) {
				record[col] = row[i]
			}
		}
		records = append(records, record)
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
