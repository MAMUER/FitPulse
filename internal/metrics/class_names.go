package metrics

var ClassNamesByID = map[int]string{
	0: "recovery",
	1: "endurance_e1e2",
	2: "threshold_e3",
	3: "strength_hiit",
}

var AllClassNames []string

func init() {
	for _, name := range ClassNamesByID {
		AllClassNames = append(AllClassNames, name)
	}
}
