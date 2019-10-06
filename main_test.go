package main

import "testing"

type MessageIDTest struct {
	Logline  string
	ID       string
	contains bool
}

var matchMessageIDTests = []MessageIDTest{
	MessageIDTest{
		`2019-09-30 06:25:30 5iExFV-0002CD-Oq => foobar@example.com R=ldap_redirect T=remote_smtp H=lists.example.org [1.2.3.44] C="250 OK id=1iEnFa-0001WR-1v`,
		"5iExFV-0002CD-Oq",
		true,
	},
}

func TestMatchMessageID(t *testing.T) {
	for _, test := range matchMessageIDTests {
		res := RegexEximMsgID.FindStringIndex(test.Logline)
		if test.contains {
			if res == nil {
				t.Errorf("Line contains message id »%s« but it is not found: »%s«", test.ID, test.Logline)
			} else {
				match := test.Logline[res[0]:res[1]]
				if match != test.ID {
					t.Errorf("Found match (»%s«) instead if (»%s«): »%s«", match, test.ID, test.Logline)
				}
			}
		}
		if !test.contains && res != nil {
			match := test.Logline[res[0]:res[1]]
			t.Errorf("Line doesn't contain a message id but one (»%s«) was found: »%s«", match, test.Logline)
		}
	}
}
