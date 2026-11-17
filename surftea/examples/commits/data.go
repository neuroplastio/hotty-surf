package main

import "time"

// A Commit is one of the repository's commits, as the list and the details
// show it.
type Commit struct {
	Hash    string
	Subject string
	Body    string
	Author  string
	When    time.Time
	Files   []File
	Tags    []Tag
}

// A File is one file a commit changed, with its added and removed lines.
type File struct {
	Path        string
	Add, Remove int
}

// A Tag is a name and a note the user put on a commit.
type Tag struct {
	Name, Note string
}

func at(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}

// demoCommits is a made-up repository's history, newest first.
func demoCommits() []*Commit {
	return []*Commit{
		{
			Hash: "9f3c2a1", Subject: "Cache parsed configs per directory",
			Body:   "Parsing the config on every lookup showed up in profiles of large\ntrees. Keep one parsed config per directory, dropped when its\nmodification time changes.",
			Author: "Mira Okafor", When: at("2026-11-14 16:42"),
			Files: []File{{"config/cache.go", 84, 0}, {"config/load.go", 12, 31}, {"config/cache_test.go", 66, 0}},
			Tags:  []Tag{{"v0.9.0", "Release candidate"}},
		},
		{
			Hash: "4b81e07", Subject: "Fix off-by-one in the pager's last page",
			Body:   "The last page dropped its final row when the row count was a\nmultiple of the page size.",
			Author: "Tomás Lindqvist", When: at("2026-11-13 10:05"),
			Files: []File{{"pager/pager.go", 3, 2}, {"pager/pager_test.go", 18, 0}},
		},
		{
			Hash: "c07d5e9", Subject: "Rename Sink to Writer",
			Body:   "Sink said nothing the io package doesn't say better.",
			Author: "Mira Okafor", When: at("2026-11-11 09:30"),
			Files: []File{{"sink.go", 0, 48}, {"writer.go", 48, 0}, {"cmd/tool/main.go", 4, 4}, {"README.md", 2, 2}},
		},
		{
			Hash: "e52a6bd", Subject: "Add a --since flag to the log command",
			Body:   "Takes a date or a duration (3d, 2w).",
			Author: "Priya Raman", When: at("2026-11-08 18:12"),
			Files: []File{{"cmd/tool/log.go", 41, 6}, {"internal/when/parse.go", 57, 0}, {"internal/when/parse_test.go", 73, 0}},
		},
		{
			Hash: "1a9f0c4", Subject: "Drop the Windows 7 workaround",
			Body:   "Nobody builds for it any more.",
			Author: "Jun Takeda", When: at("2026-11-05 11:47"),
			Files: []File{{"term/console_windows.go", 0, 62}},
		},
		{
			Hash: "7d2e8f3", Subject: "Document the plugin interface",
			Body:   "",
			Author: "Priya Raman", When: at("2026-11-02 14:20"),
			Files: []File{{"docs/plugins.md", 118, 9}, {"plugin/plugin.go", 22, 3}},
		},
		{
			Hash: "b3640aa", Subject: "Initial import",
			Body:   "",
			Author: "Jun Takeda", When: at("2026-10-28 08:00"),
			Files: []File{{"go.mod", 3, 0}, {"main.go", 25, 0}, {"README.md", 7, 0}},
			Tags:  []Tag{{"v0.1.0", ""}},
		},
	}
}
