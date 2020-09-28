package backend

import (
	"regexp"
	"testing"
)

var (
	SupportedCommandList = []string{
		"SELECT * FROM cpu",
		"SELECT * FROM \"cpu\"",
		"SELECT xx FROM cpu WHERE region='uswest' AND price=10",
		"SELECT mean(\"value\") FROM \"cpu\" WHERE \"region\" = 'uswest' GROUP BY time(10m) fill(0)",
		"SHOW FIELD KEYS FROM cpu",
		"SHOW FIELD KEYS FROM \"cpu\"",
		"SHOW SERIES FROM \"telegraf\".\"autogen\".\"cpu\" WHERE cpu = 'cpu8'",
		"SHOW TAG KEYS FROM cpu",
		"SHOW TAG KEYS FROM \"cpu\" WHERE \"region\" = 'uswest'",
		"SHOW TAG KEYS FROM cpu WHERE \"host\" = 'serverA'",
		"SHOW TAG VALUES FROM cpu WITH KEY = \"region\"",
		"SHOW TAG VALUES FROM \"1h\".\"cpu\" WITH KEY = \"region\"",
		"SHOW TAG VALUES FROM cpu WITH KEY !~ /.*c.*/",
		"SHOW TAG VALUES FROM \"cpu\" WITH KEY IN (\"region\", \"host\") WHERE \"service\" = 'redis'",
		"SHOW FIELD KEYS FROM \"1h\".\"cpu\"",
		"SHOW FIELD KEYS FROM \"cpu.load\"",
		"SHOW FIELD KEYS FROM \"1h\".\"cpu.load\"",
	}
	UnsupportedCommandList = []string{
		"REVOKE ALL PRIVILEGES FROM \"jdoe\"",
		"REVOKE READ ON \"mydb\" FROM \"jdoe\"",
		"DELETE FROM \"cpu\"",
		"DELETE FROM \"cpu\" WHERE time < '2000-01-01T00:00:00Z'",
		"DROP SERIES FROM \"telegraf\".\"autogen\".\"cpu\" WHERE cpu = 'cpu8'",
	}
)

func TestSupportedCmds(t *testing.T) {
	r, _ := regexp.Compile(SupportCommands)
	for _, cmd := range SupportedCommandList {
		if !r.MatchString(cmd) {
			t.Errorf("Error testing supported cmd: %s", cmd)
		}
	}
}

func TestUnsupportedCmds(t *testing.T) {
	r, _ := regexp.Compile(ForbidCommands)
	for _, cmd := range UnsupportedCommandList {
		if !r.MatchString(cmd) {
			t.Errorf("Error testing supported cmd: %s", cmd)
		}
	}
}
