package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

func main() {
	for i := 15; i <= 18; i++ {
		for _, pos := range positions {
			fileName := fmt.Sprintf("weekly-proj-%s", pos)
			if pos == "QB" {
				fileName = "week-proj"
			}
			path := fmt.Sprintf(`nfl/%s (%d).csv`, fileName, i)
			data, gw, err := readAndParseCsv(path)
			if err != nil {
				panic(fmt.Sprintf("error while handling csv file: %s\n", err))
			}
			json, err := csvToJson(data)
			if err != nil {
				panic(fmt.Sprintf("error while converting csv to json file: %s\n", err))
			}
			err = os.WriteFile(fmt.Sprintf("nfl-rotowire/%s_gameweek_%d.json", pos, gw), []byte(json), 0644)
			if err != nil {
				panic(fmt.Sprintf("error while writing json file: %s\n", err))
			}
		}
	}
}

var positions = map[int]string{
	0: "QB",
	1: "RB",
	2: "WR",
	3: "TE",
	4: "K",
	5: "def",
}

func readAndParseCsv(path string) ([][]string, int, error) {
	csvFile, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening %s", path)
	}

	var rows [][]string

	reader := csv.NewReader(csvFile)
	row, _ := reader.Read() // skip header
	gameweek := 2
	for _, elem := range row {
		if strings.Contains(elem, "Week") {
			split := strings.Split(elem, " ")[1]
			gameweek, err = strconv.Atoi(split)
			if err != nil {
				return rows, 0, fmt.Errorf("failed to parse gameweek: %s", err)
			}
		}
	}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return rows, 0, fmt.Errorf("failed to parse csv: %s", err)
		}

		rows = append(rows, row)
	}

	return rows, gameweek, nil
}

func csvToJson(rows [][]string) (string, error) {
	var entries []map[string]interface{}
	attributes := rows[0]
	for _, row := range rows[1:] {
		entry := map[string]interface{}{}
		for i, value := range row {
			attribute := attributes[i]
			attribute = strcase.ToLowerCamel(attribute)
			// split csv header key for nested objects
			objectSlice := strings.Split(attribute, ".")
			internal := entry
			for index, val := range objectSlice {
				// split csv header key for array objects
				key, arrayIndex := arrayContentMatch(val)
				if arrayIndex != -1 {
					if internal[key] == nil {
						internal[key] = []interface{}{}
					}
					internalArray := internal[key].([]interface{})
					if index == len(objectSlice)-1 {
						internalArray = append(internalArray, value)
						internal[key] = internalArray
						break
					}
					if arrayIndex >= len(internalArray) {
						internalArray = append(internalArray, map[string]interface{}{})
					}
					internal[key] = internalArray
					internal = internalArray[arrayIndex].(map[string]interface{})
				} else {
					if index == len(objectSlice)-1 {
						if val, err := strconv.ParseFloat(value, 64); err == nil {
							internal[key] = val
							break
						}
						internal[key] = value
						break
					}
					if internal[key] == nil {
						internal[key] = map[string]interface{}{}
					}
					internal = internal[key].(map[string]interface{})
				}
			}
		}
		entries = append(entries, entry)
	}
	projections := struct {
		Projections []map[string]interface{} `json:"projections"`
	}{
		Projections: entries,
	}
	bytes, err := json.MarshalIndent(projections, "", "	")
	if err != nil {
		return "", fmt.Errorf("marshal error %s", err)
	}

	return string(bytes), nil
}

func arrayContentMatch(str string) (string, int) {
	i := strings.Index(str, "[")
	if i >= 0 {
		j := strings.Index(str, "]")
		if j >= 0 {
			index, _ := strconv.Atoi(str[i+1 : j])
			return str[0:i], index
		}
	}
	return str, -1
}
