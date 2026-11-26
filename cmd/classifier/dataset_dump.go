package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// dumpDatasetFlag извлекает -dump-dataset <path> из аргументов запуска.
// Флаг не поднимает сервер: classifier-DATESET <path> завершается сразу.
func dumpDatasetFlag(args []string) string {
	for i, arg := range args {
		if arg == "-dump-dataset" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "-dump-dataset=") {
			return strings.TrimPrefix(arg, "-dump-dataset=")
		}
	}
	return ""
}

// distillAge — опорный возраст, для которого строится обучающая выборка.
// Модель линейна и не получает возраст на входе, поэтому ЧСС канонизируется
// к этому возрасту (см. canonicalHeartRate): на инференсе ЧСС пользователя
// пересчитывается к тому же опорному max HR, иначе rule-based и ML расходятся.
const distillAge = 30

// canonicalHeartRate приводит ЧСС пользователя к шкале опорного возраста.
// 220-30 = 190 — опорный HRmax; 220-age — фактический HRmax пользователя.
//
// Функция оставлена как часть контракта датасета: она документирует, как
// ЧСС пересчитывается к distillAge. Используется в тестах консистентности.
func canonicalHeartRate(hr float64, age int) float64 {
	if age <= 0 {
		return hr
	}
	hrMax := 220.0 - float64(age)
	if hrMax <= 0 {
		return hr
	}
	const refHRMax = 190.0
	return hr * refHRMax / hrMax
}

// distillArchetype описывает центр выборки для одного функционального класса.
// Разброс задаётся долей от диапазона, чтобы линейная модель видела плавные
// границы, а не только идеальные точки.
type distillArchetype struct {
	name      string
	heartRate [2]float64
	hrv       [2]float64
	spo2      [2]float64
	temp      [2]float64
	sleep     [2]float64
}

var distillArchetypes = []distillArchetype{
	{"recovery", [2]float64{60, 118}, [2]float64{55, 100}, [2]float64{96.5, 99.5}, [2]float64{36.0, 36.8}, [2]float64{6.5, 9.5}},
	{"endurance_basic", [2]float64{126, 150}, [2]float64{35, 60}, [2]float64{95.5, 98.5}, [2]float64{36.4, 37.0}, [2]float64{6.0, 9.0}},
	{"endurance_threshold", [2]float64{153, 169}, [2]float64{22, 40}, [2]float64{94.5, 97.0}, [2]float64{36.6, 37.2}, [2]float64{5.5, 8.5}},
	{"power_hiit", [2]float64{172, 198}, [2]float64{8, 24}, [2]float64{93.5, 96.5}, [2]float64{36.8, 37.5}, [2]float64{5.0, 8.0}},
	{"overtraining", [2]float64{85, 112}, [2]float64{8, 26}, [2]float64{95.5, 99.0}, [2]float64{36.2, 37.0}, [2]float64{2.5, 4.6}},
	{"illness", [2]float64{88, 125}, [2]float64{8, 26}, [2]float64{91.0, 95.0}, [2]float64{38.1, 39.6}, [2]float64{4.0, 8.0}},
	{"unknown", [2]float64{95, 145}, [2]float64{20, 55}, [2]float64{89.5, 92.8}, [2]float64{36.2, 37.2}, [2]float64{4.5, 8.5}},
}

func jitter(rng *rand.Rand, bounds [2]float64) float64 {
	return bounds[0] + rng.Float64()*(bounds[1]-bounds[0])
}

// dumpDistilledDataset генерирует CSV с эталонными состояниями, размеченными
// rule-based классификатором. Это «учитель» для линейной аппроксимации:
// модель обучается на той же предметной логике, что работает в проде, поэтому
// ML-путь не может расходиться с правилами на базовой выборке.
func dumpDistilledDataset(path string, perClass int) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		// #nosec G703 -- путь задаётся флагом запуска
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create dataset dir: %w", err)
		}
	}

	// #nosec G304,G703 -- путь задаётся флагом запуска
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create dataset: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"heart_rate", "hrv", "spo2", "temperature",
		"systolic_pressure", "diastolic_pressure", "sleep_hours", "label",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// #nosec G404 -- детерминированная выборка для воспроизводимости
	rng := rand.New(rand.NewSource(20261122))
	counts := make([]int, len(trainingClasses))
	written := 0

	for _, arch := range distillArchetypes {
		for i := 0; i < perClass; i++ {
			data := physiologicalData{
				HeartRate:              jitter(rng, arch.heartRate),
				HeartRateVariability:   jitter(rng, arch.hrv),
				SpO2:                   jitter(rng, arch.spo2),
				Temperature:            jitter(rng, arch.temp),
				BloodPressureSystolic:  100 + rng.Float64()*50,
				BloodPressureDiastolic: 55 + rng.Float64()*45,
				SleepHours:             jitter(rng, arch.sleep),
			}

			class, _, _ := classifyState(data, distillAge)
			if class < 0 || class >= len(counts) {
				continue
			}
			counts[class]++

			row := []string{
				strconv.FormatFloat(data.HeartRate, 'f', 3, 64),
				strconv.FormatFloat(data.HeartRateVariability, 'f', 3, 64),
				strconv.FormatFloat(data.SpO2, 'f', 3, 64),
				strconv.FormatFloat(data.Temperature, 'f', 3, 64),
				strconv.FormatFloat(data.BloodPressureSystolic, 'f', 3, 64),
				strconv.FormatFloat(data.BloodPressureDiastolic, 'f', 3, 64),
				strconv.FormatFloat(data.SleepHours, 'f', 3, 64),
				strconv.Itoa(class),
			}
			if err := w.Write(row); err != nil {
				return fmt.Errorf("write row: %w", err)
			}
			written++
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush dataset: %w", err)
	}

	fmt.Printf("Дамп датасета: %s (строк: %d)\n", path, written)
	for class, count := range counts {
		fmt.Printf("  класс %d (%s): %d\n", class, trainingClasses[class].Name, count)
	}
	return nil
}
