#!/bin/bash
# scripts/run_full_training.sh

echo "=============================================="
echo "ПОЛНОЕ ОБУЧЕНИЕ НА ВСЕХ ДАТАСЕТАХ"
echo "=============================================="

echo ""
echo "Шаг 3: Обучение GAN-генератора..."
cd ../ml_generator || exit
python train_gan.py

echo ""
echo "=============================================="
echo "ВСЕ МОДЕЛИ ОБУЧЕНЫ!"
echo "=============================================="
echo ""
echo "Модели: ../../models/"
echo "   - classifier.keras"
echo "   - generator.keras"
echo "   - scaler.pkl"
echo ""
echo "Статистика: ../../datasets/processed/"
echo "   - training_data_real.csv"
echo "   - dataset_stats.json"
echo "   - preprocessing_log.json"
echo ""
