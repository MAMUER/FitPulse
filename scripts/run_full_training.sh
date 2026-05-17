#!/bin/bash
# scripts/run_full_training.sh

echo "=============================================="
echo "🚀 ПОЛНОЕ ОБУЧЕНИЕ НА ВСЕХ ДАТАСЕТАХ"
echo "=============================================="

cd cmd/ml-classifier

echo ""
echo "📊 Шаг 1: Препроцессинг ВСЕХ датасетов..."
echo "   (Это может занять 10-60 минут в зависимости от объема)"
python preprocess_real_data.py

echo ""
echo "🧠 Шаг 2: Обучение классификатора..."
python train.py

echo ""
echo "🎨 Шаг 3: Обучение GAN-генератора..."
cd ../ml-generator
python train_gan.py

echo ""
echo "=============================================="
echo "ВСЕ МОДЕЛИ ОБУЧЕНЫ!"
echo "=============================================="
echo ""
echo "📁 Модели: ../../models/"
echo "   - classifier.keras"
echo "   - generator.keras"
echo "   - scaler.pkl"
echo ""
echo "📊 Статистика: ../../datasets/processed/"
echo "   - training_data_real.csv"
echo "   - dataset_stats.json"
echo "   - preprocessing_log.json"
echo ""