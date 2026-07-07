#!/bin/bash
# scripts/run_full_training.sh

echo "=============================================="
echo "ПОЛНОЕ ОБУЧЕНИЕ НА ВСЕХ ДАТАСЕТАХ"
echo "=============================================="

echo ""
echo "Шаг 1: Обучение Diffusion-генератора..."
cd ../ml_generator || exit
python train_gan.py

echo ""
echo "=============================================="
echo "ВСЕ МОДЕЛИ ОБУЧЕНЫ!"
echo "=============================================="
echo ""
echo "Модели: ../../models/"
echo "   - generator.pt (PyTorch)"
echo "   - generator.onnx (ONNX Runtime)"
echo ""
echo "Статистика: ../../datasets/processed/"
echo "   - training_plans_exercises.csv"
echo ""
echo "Логи: wandb.ai/fitpulse/fitpulse-generator"
echo ""