package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

type AppConfig struct {
	SearchString                string
	SearchStringIsRegex         bool
	SearchStringIsCaseSensitive bool
	EximFormatOnly              bool
}

var (
	Config                      AppConfig
	RegexEximMsgID, RegexSetEnd *regexp.Regexp
)

func init() {
	RegexEximMsgID = regexp.MustCompile(`\b[0-9A-Za-z]{6}-[0-9A-Za-z]{6}-[0-9A-Za-z]{2}\b`)
	RegexSetEnd = regexp.MustCompile(`Completed|SMTP data timeout|rejected`)

	flag.BoolVar(&Config.SearchStringIsRegex, "r", false, "Interpret search string as regular expression instead of fixed string")
	flag.BoolVar(&Config.SearchStringIsCaseSensitive, "c", false, "Search string is case sensitive")
	//flag.BoolVar(&Config.EximFormatOnly, "e", false, "Only match exim logfiles (only match exim timestamps instead of looking for an isolated message id string)")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Please provide a search string")
	} else {
		Config.SearchString = flag.Args()[0]
	}
}

type Match struct {
	Lines       []string
	Interesting bool
}

type MatchLine func(*string) bool
type MatchMessageID func(*string) (bool, string)

func main() {
	var (
		input io.Reader
	)

	// prepare line matcher
	var matcher MatchLine
	if Config.SearchStringIsRegex {
		var (
			re       *regexp.Regexp
			restring string
			err      error
		)
		if Config.SearchStringIsCaseSensitive {
			restring = Config.SearchString
		} else {
			restring = `(?i:` + Config.SearchString + `)`
		}
		re, err = regexp.Compile(restring)
		if err != nil {
			log.Fatalf("Error compiling search string »%s« as regular expression: %s", restring, err)
		}
		matcher = func(in *string) bool { return re.MatchString(*in) }
	} else {
		if Config.SearchStringIsCaseSensitive {
			matcher = func(in *string) bool { return strings.Contains(*in, Config.SearchString) }
		} else {
			searchStringLC := strings.ToLower(Config.SearchString)
			matcher = func(in *string) bool { return strings.Contains(strings.ToLower(*in), strings.ToLower(searchStringLC)) }
		}
	}

	// prepare message id matcher
	var msgidmatcher MatchMessageID
	// generic regex version
	msgidmatcher = func(in *string) (bool, string) {
		loc := RegexEximMsgID.FindStringIndex(*in)
		if loc == nil {
			return false, ""
		} else {
			return true, (*in)[loc[0]:loc[1]]
		}
	}

	// prepare inputs
	if flag.NArg() == 1 {
		input = os.Stdin
	} else {
		var infiles []io.Reader
		for _, filename := range flag.Args()[1:] {
			reader, err := os.Open(filename)
			if err == nil {
				infiles = append(infiles, reader)
			} else {
				log.Printf("Unable to open file %s: %s", filename, err)
			}
		}
		input = io.MultiReader(infiles...)
	}

	// search input
	var (
		LinesByMsgID = make(map[string]*Match)
	)
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		// find message id
		hasID, id := msgidmatcher(&line)
		// message has message ID
		if hasID {
			// message id known?
			match, exists := LinesByMsgID[id]
			if exists {
				// add line to existing struct
				match.Lines = append(match.Lines, line)
			} else {
				// create new struct
				match = &Match{Lines: []string{line}, Interesting: false}
				LinesByMsgID[id] = match
			}
			// does the line contain the search string?
			if matcher(&line) {
				match.Interesting = true
			}
			// is line the last line in the set?
			if RegexSetEnd.MatchString(line) {
				// only print interesting sets
				if match.Interesting {
					for _, l := range match.Lines {
						fmt.Println(l)
					}
					fmt.Println("")
				}
				delete(LinesByMsgID, id)
			}
		// matching line without associated message id
		} else {
			if matcher(&line) {
				fmt.Println(line)
			}
		}
	}
}
