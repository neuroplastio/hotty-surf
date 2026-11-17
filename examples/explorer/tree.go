package main

import "time"

// project is the made-up project the app browses, in memory: nothing
// here is on the disk.
func project() Entry {
	id := 0
	day := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	file := func(name string, size int64, days, mins int) Entry {
		id++
		return Entry{ID: id, Name: name, Size: size, Modified: day.Add(time.Duration(days)*24*time.Hour + time.Duration(mins)*time.Minute)}
	}
	dir := func(name string, days int, children ...Entry) Entry {
		e := file(name, 0, days, 0)
		e.Dir, e.Children = true, children
		return e
	}
	root := dir("lumen", 70,
		dir("cmd", 64,
			dir("lumen", 64,
				file("main.go", 1840, 64, 312),
				file("flags.go", 2210, 41, 95),
			),
		),
		dir("docs", 68,
			file("guide.md", 14_820, 68, 410),
			file("install.md", 3_105, 12, 60),
			file("themes.md", 6_402, 55, 220),
			dir("images", 30,
				file("hero.png", 284_311, 30, 15),
				file("layout.svg", 9_870, 22, 40),
			),
		),
		dir("internal", 66,
			dir("build", 66,
				file("build.go", 18_342, 66, 140),
				file("build_test.go", 9_771, 66, 150),
				file("markdown.go", 7_015, 47, 500),
			),
			dir("serve", 59,
				file("serve.go", 6_480, 59, 85),
				file("watch.go", 3_390, 38, 30),
			),
		),
		dir("themes", 50,
			dir("paper", 50,
				file("base.html", 2_048, 50, 10),
				file("style.css", 11_604, 52, 300),
			),
		),
		file("go.mod", 412, 63, 10),
		file("go.sum", 5_930, 63, 11),
		file("LICENSE", 11_357, 0, 0),
		file("Makefile", 980, 45, 200),
		file("README.md", 4_216, 70, 480),
	)
	return root
}
