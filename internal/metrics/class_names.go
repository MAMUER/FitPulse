package metrics

var ClassNamesByID = map[int]string{
	0: "recovery",
	1: "endurance_basic",
	2: "endurance_threshold",
	3: "power_hiit",
	4: "overtraining",
	5: "illness",
}

var AllClassNames []string

func init() {
	for _, name := range ClassNamesByID {
		AllClassNames = append(AllClassNames, name)
	}
}
