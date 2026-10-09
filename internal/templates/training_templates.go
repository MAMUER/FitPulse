// Package templates provides training templates for plan generation.
package templates

type TrainingTemplate struct {
	NameRu         string
	DurationRange  [2]int
	IntensityRange [2]float64
	RestRatio      float64
	Exercises      []string
}

var TrainingTemplates = map[string]TrainingTemplate{
	"recovery": {
		NameRu:         "Восстановление",
		DurationRange:  [2]int{20, 40},
		IntensityRange: [2]float64{0.3, 0.5},
		RestRatio:      0.7,
		Exercises:      []string{"лёгкая разминка", "растяжка", "дыхательные упражнения", "йога"},
	},
	"endurance_basic": {
		NameRu:         "Базовая выносливость",
		DurationRange:  [2]int{45, 90},
		IntensityRange: [2]float64{0.5, 0.7},
		RestRatio:      0.4,
		Exercises:      []string{"бег", "велосипед", "плавание", "лыжи"},
	},
	"endurance_threshold": {
		NameRu:         "Пороговая выносливость",
		DurationRange:  [2]int{30, 60},
		IntensityRange: [2]float64{0.7, 0.85},
		RestRatio:      0.3,
		Exercises:      []string{"темновой бег", "интервалы на пороге", "фартлек", "критическая мощность"},
	},
	"power_hiit": {
		NameRu:         "Силовая/HIIT",
		DurationRange:  [2]int{20, 45},
		IntensityRange: [2]float64{0.85, 1.0},
		RestRatio:      0.5,
		Exercises:      []string{"HIIT", "силовые", "спринты", "кроссфит"},
	},
	"overtraining": {
		NameRu:         "Перетренированность",
		DurationRange:  [2]int{0, 20},
		IntensityRange: [2]float64{0.0, 0.3},
		RestRatio:      0.8,
		Exercises:      []string{"отдых", "ходьба", "растяжка", "йога"},
	},
	"illness": {
		NameRu:         "Заболевание",
		DurationRange:  [2]int{0, 0},
		IntensityRange: [2]float64{0.0, 0.0},
		RestRatio:      1.0,
		Exercises:      []string{"полный отдых"},
	},
}

func GetTemplate(classification string) (TrainingTemplate, bool) {
	tpl, ok := TrainingTemplates[classification]
	return tpl, ok
}
