package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const configFilename string = "config.json"
const suffixRegex string = "[\"'` \\n]"
const prefixRegex string = "(?i)[\"'` \\n]"

type Config struct {
	Ignore   []string
	Include  []string
	Comments []string
	Literals []string
	Keywords []string
}

type stringWithPosition struct {
	lineNumber int
	position   []int
	text       string
}

func main() {
	//todo defer
	argsWithoutExecutable := os.Args[1:]
	var parsingPath string
	if len(argsWithoutExecutable) == 0 {
		parsingPath, _ = os.Getwd()
	}
	fmt.Println(parsingPath)

	config := loadConfig()

	file, err := os.Open(parsingPath)
	if err != nil {
		// handle the error and return TODO
	}
	defer file.Close()

	// This returns an *os.FileInfo type
	fileInfo, err := file.Stat()
	if err != nil {
		// error handling
	}

	// IsDir is short for fileInfo.Mode().IsDir()
	if fileInfo.IsDir() {
		excludeRegex := regexp.MustCompile("^(" + strings.Join(config.Ignore, "|") + ")$")
		includeRegex := regexp.MustCompile(".*(" + strings.Join(config.Include, "|") + ")$")

		err := filepath.Walk(parsingPath,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				fileName := info.Name()

				isIncluded := includeRegex.MatchString(fileName)
				if !info.IsDir() && isIncluded {
					sqlQueries := parseFile(path, config)
					printResult(parsingPath, sqlQueries)
					return nil
				}

				isExcluded := excludeRegex.MatchString(fileName)
				if info.IsDir() && isExcluded {
					return filepath.SkipDir
				}

				return nil
			})
		if err != nil {
			log.Println(err)
		}

	} else {
		sqlQueries := parseFile(parsingPath, config)
		printResult(parsingPath, sqlQueries)
	}
}

func loadConfig() Config {
	configData, err := os.ReadFile(configFilename)
	if err != nil {
		fmt.Print("Can't read config file: ", configFilename, err)
		panic(err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Print("Incorrect config file content: ", configFilename, err)
		panic(err)
	}
	return config
}

func parseFile(filename string, config Config) []stringWithPosition {
	binaryData, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print("Source file reading error: ", filename, err)
		panic(err)
	}

	text := string(binaryData)

	sqlQueries := getExtractedInformation(text, config)
	lineBrakes := getLineBrakesIndexes(text)
	setLineNumbers(lineBrakes, sqlQueries)
	return sqlQueries
}

func getExtractedInformation(text string, config Config) []stringWithPosition {
	comments := getRegexMatchPositions(text, config.Comments)
	stringLiterals := getRegexMatchPositions(text, config.Literals)

	var keywordRegexCollection []*regexp.Regexp

	for _, keyword := range config.Keywords {
		regexString := prefixRegex + keyword + suffixRegex //todo
		regexp := regexp.MustCompile(regexString)
		keywordRegexCollection = append(keywordRegexCollection, regexp)
	}

	var sqlQueries []stringWithPosition

	for _, literalRange := range stringLiterals {
		literalString := text[literalRange[0]:literalRange[1]]

		//TODO оптимизировать
		isSql := isSql(literalString, keywordRegexCollection)
		isComment := isComment(comments, literalRange)

		if isSql && !isComment {
			sqlQueries = append(sqlQueries, stringWithPosition{-1, literalRange, literalString})
		}
		fmt.Println(isSql, " ", literalString)
	}
	return sqlQueries
}

func getLineBrakesIndexes(text string) []int {
	var lineBrakes []int
	for i, ch := range text {
		if ch == '\n' {
			lineBrakes = append(lineBrakes, i)
		}
	}
	return lineBrakes
}

func isComment(comments [][]int, literalRange []int) bool {
	literalStart := literalRange[0]
	var result bool
	for _, commentRange := range comments {
		if commentRange[0] < literalStart &&
			literalStart < commentRange[1] {
			result = true
			break
		}
	}
	return result
}

func setLineNumbers(lineBrakes []int, sqlQueries []stringWithPosition) {
	for ind, sqlQuery := range sqlQueries {
		for i, pos := range lineBrakes {
			if pos > sqlQuery.position[0] {
				sqlQueries[ind].lineNumber = i + 1
				break
			}
		}
	}
}

func printResult(filename string, sqlQueries []stringWithPosition) {
	for _, sqlQuery := range sqlQueries {
		fmt.Println(filename, ",", sqlQuery.lineNumber, ",", sqlQuery.text)
	}
}

func isOneWord(literalString string) bool {
	separators := []rune{' ', '\t', '\n'}
	for _, letter := range literalString {
		for _, separator := range separators {
			if letter == separator {
				return false
			}
		}
	}
	return true
}

func getRegexMatchPositions(text string, commentsRegex []string) [][]int {
	var comments [][]int
	for _, commentRegex := range commentsRegex {
		regex := regexp.MustCompile(commentRegex)
		result := regex.FindAllStringIndex(text, -1)
		comments = append(comments, result...)
	}
	return comments
}

func isSql(literalString string, keywordRegexCollection []*regexp.Regexp) bool {
	if isOneWord(literalString) {
		return false
	}

	for _, keywordRegexp := range keywordRegexCollection {
		if keywordRegexp.FindString(literalString) != "" {
			fmt.Println(keywordRegexp.String())
			return true
		}
	}

	return false
}
