# cmd/ml-classifier/explore_datasets.py
"""
Dataset Exploration Script - ДИНАМИЧЕСКИЙ ПОИСК по симлинкам
"""
import os
import pandas as pd
import numpy as np
import pickle
import json
from datetime import datetime
from pathlib import Path

RAW_DATA_DIR = '../../datasets/raw'
OUTPUT_DIR = '../../datasets/processed'
REPORT_FILE = f'{OUTPUT_DIR}/dataset_exploration_report.json'

class DatasetExplorer:
    def __init__(self, raw_data_dir):
        self.raw_data_dir = raw_data_dir
        self.report = {
            'timestamp': datetime.now().isoformat(),
            'datasets': {},
            'summary': {},
            'recommendations': []
        }
    
    def explore_all(self):
        print("=" * 80)
        print("DATASET EXPLORATION — ЧЕРЕЗ СИМЛИНКИ")
        print("=" * 80)
        print(f"Raw data directory: {self.raw_data_dir}")
        print(f"Directory exists: {os.path.exists(self.raw_data_dir)}")
        
        if not os.path.exists(self.raw_data_dir):
            print("ERROR: Raw data directory not found!")
            return self.report
        
        available_folders = []
        for item in os.listdir(self.raw_data_dir):
            item_path = os.path.join(self.raw_data_dir, item)
            if os.path.isdir(item_path):
                available_folders.append(item)
        
        print(f"\nНАЙДЕНО {len(available_folders)} папок через симлинки:")
        for folder in sorted(available_folders):
            print(f"  • {folder}")
        
        # Вызываем исследование для каждой папки
        self.explore_by_folder(available_folders)
        self.generate_summary()
        self.save_report()
        
        return self.report
    
    def explore_by_folder(self, folders):
        """Исследуем каждую папку динамически"""
        
        for folder in folders:
            folder_path = os.path.join(self.raw_data_dir, folder)
            print(f"\n{'=' * 80}")
            print(f"{folder.upper()}")
            print(f"{'=' * 80}")
            
            dataset_info = {
                'name': folder,
                'path': folder_path,
                'subjects': 0,
                'files': [],
                'signals': [],
                'sample_data': {}
            }
            
            # Считаем файлы и подпапки
            try:
                items = os.listdir(folder_path)
                subdirs = [d for d in items if os.path.isdir(os.path.join(folder_path, d))]
                files = [f for f in items if os.path.isfile(os.path.join(folder_path, f))]
                
                dataset_info['total_files'] = len(files)
                dataset_info['total_subdirs'] = len(subdirs)
                
                print(f"Папка доступна: {len(files)} файлов, {len(subdirs)} подпапок")
                
                # Определяем тип датасета по структуре
                if folder == 'bidmc':
                    self._explore_bidmc(folder_path, dataset_info)
                elif folder == 'wesad':
                    self._explore_wesad(folder_path, dataset_info)
                elif folder == 'spd':
                    self._explore_spd(folder_path, dataset_info)
                elif folder == 'wesd':
                    self._explore_wesd(folder_path, dataset_info)
                elif folder.startswith('capnobase'):
                    self._explore_capnobase(folder, folder_path, dataset_info)
                elif folder == 'ppg_dalia':
                    self._explore_ppg_dalia(folder_path, dataset_info)
                elif folder == 'sleep_edf':
                    self._explore_sleep_edf(folder_path, dataset_info)
                elif folder == 'stress_nurses':
                    self._explore_stress_nurses(folder_path, dataset_info)
                elif folder == 'e4selflearning':
                    self._explore_e4selflearning(folder_path, dataset_info)
                elif folder == 'big_ideas_lab':
                    self._explore_big_ideas_lab(folder_path, dataset_info)
                elif folder == 'adarp':
                    self._explore_adarp(folder_path, dataset_info)
                elif folder == 'weee':
                    self._explore_weee(folder_path, dataset_info)
                elif folder == 'toadstool':
                    self._explore_toadstool(folder_path, dataset_info)
                elif folder == 'ue4w':
                    self._explore_ue4w(folder_path, dataset_info)
                elif folder == 'csl':
                    self._explore_csl(folder_path, dataset_info)
                else:
                    print(f" Неизвестный формат датасета")
                    dataset_info['files'] = files[:10]
                
                self.report['datasets'][folder] = dataset_info
                
            except Exception as e:
                print(f"Ошибка при исследовании {folder}: {e}")
                dataset_info['error'] = str(e)
                self.report['datasets'][folder] = dataset_info
    
    def _explore_bidmc(self, path, info):
        """BIDMC: bidmc_XX_Numerics.csv"""
        files = os.listdir(path)
        numerics = [f for f in files if 'Numerics.csv' in f]
        info['subjects'] = len(numerics)
        info['files'] = numerics[:5]
        print(f"BIDMC: {len(numerics)} субъектов (Numerics файлы)")
        
        if numerics:
            sample = os.path.join(path, numerics[0])
            df = pd.read_csv(sample, nrows=3)
            info['signals'] = list(df.columns)[:10]
            print(f"📡 Сигналы: {info['signals'][:5]}...")
    
    def _explore_wesad(self, path, info):
        """WESAD: S2, S3... с .pkl"""
        subjects = [d for d in os.listdir(path) if d.startswith('S') and os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        print(f"WESAD: {len(subjects)} субъектов")
        
        if subjects:
            subj_path = os.path.join(path, subjects[0])
            pkl_files = [f for f in os.listdir(subj_path) if f.endswith('.pkl')]
            if pkl_files:
                pkl_path = os.path.join(subj_path, pkl_files[0])
                with open(pkl_path, 'rb') as f:
                    data = pickle.load(f, encoding='latin1')
                if isinstance(data, dict) and 'signal' in data:
                    for sensor in ['chest', 'wrist']:
                        if sensor in data['signal']:
                            signals = list(data['signal'][sensor].keys())
                            info['signals'].extend(signals)
                            print(f"📡 {sensor}: {signals[:5]}")
    
    def _explore_spd(self, path, info):
        """SPD: S01-S35 с CSV"""
        subjects = [d for d in os.listdir(path) if d.startswith('S') and os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        print(f"SPD: {len(subjects)} субъектов")
        
        if subjects:
            subj_path = os.path.join(path, subjects[0])
            csv_files = [f for f in os.listdir(subj_path) if f.endswith('.csv')]
            info['files'] = csv_files
            info['signals'] = [f.split('.')[0] for f in csv_files]
            print(f"📡 Сигналы: {info['signals']}")
    
    def _explore_wesd(self, path, info):
        """WESD: S1-S10 с сессиями"""
        subjects = [d for d in os.listdir(path) if d.startswith('S') and os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        print(f"WESD: {len(subjects)} субъектов")
        
        if subjects:
            subj_path = os.path.join(path, subjects[0])
            sessions = [d for d in os.listdir(subj_path) if os.path.isdir(os.path.join(subj_path, d))]
            info['sessions'] = sessions
            print(f"📚 Сессии: {sessions}")
            
            if sessions:
                session_path = os.path.join(subj_path, sessions[0])
                csv_files = [f for f in os.listdir(session_path) if f.endswith('.csv')]
                info['signals'] = [f.split('.')[0] for f in csv_files]
                print(f"📡 Сигналы: {info['signals']}")
    
    def _explore_capnobase(self, name, path, info):
        """CapnoBase: meta.csv, param.csv"""
        files = [f for f in os.listdir(path) if f.endswith(('.csv', '.tab'))]
        info['files'] = len(files)
        print(f"{name}: {len(files)} файлов")
        
        if files:
            sample = os.path.join(path, files[0])
            try:
                if sample.endswith('.tab'):
                    df = pd.read_csv(sample, sep='\t', nrows=3)
                else:
                    df = pd.read_csv(sample, nrows=3)
                info['signals'] = list(df.columns)[:10]
                print(f"📡 Колонки: {info['signals'][:5]}...")
            except:
                pass
    
    def _explore_ppg_dalia(self, path, info):
        subjects = [d for d in os.listdir(path) if d.startswith('S') and os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        print(f"PPG DaLiA: {len(subjects)} субъектов")
    
    def _explore_sleep_edf(self, path, info):
        records = [f for f in os.listdir(path) if f.endswith('.rec')]
        info['records'] = len(records)
        print(f"Sleep EDF: {len(records)} записей")
    
    def _explore_stress_nurses(self, path, info):
        subjects = [d for d in os.listdir(path) if os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        total_zips = 0
        for subj in subjects:
            zips = [f for f in os.listdir(os.path.join(path, subj)) if f.endswith('.zip')]
            total_zips += len(zips)
        info['samples'] = total_zips
        print(f"Stress Nurses: {len(subjects)} субъектов, {total_zips} сэмплов")
    
    def _explore_e4selflearning(self, path, info):
        wearable_path = os.path.join(path, 'class_wearable_data')
        if os.path.exists(wearable_path):
            classes = [d for d in os.listdir(wearable_path) if os.path.isdir(os.path.join(wearable_path, d))]
            info['participants'] = len(classes)
            print(f"E4 Self-Learning: {len(classes)} классов")
    
    def _explore_big_ideas_lab(self, path, info):
        subjects = [d for d in os.listdir(path) if d.isdigit() and os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        print(f"Big Ideas Lab: {len(subjects)} субъектов")
    
    def _explore_adarp(self, path, info):
        parts = [d for d in os.listdir(path) if d.startswith('Part') and os.path.isdir(os.path.join(path, d))]
        info['parts'] = len(parts)
        print(f"ADARP: {len(parts)} частей")
    
    def _explore_weee(self, path, info):
        subjects = [d for d in os.listdir(path) if d.startswith('P') and os.path.isdir(os.path.join(path, d))]
        info['subjects'] = len(subjects)
        print(f"WEEE: {len(subjects)} субъектов")
    
    def _explore_toadstool(self, path, info):
        participants_path = os.path.join(path, 'participants')
        if os.path.exists(participants_path):
            participants = [d for d in os.listdir(participants_path) if d.startswith('participant_')]
            info['participants'] = len(participants)
            print(f"Toadstool: {len(participants)} участников")
    
    def _explore_ue4w(self, path, info):
        sessions = [d for d in os.listdir(path) if os.path.isdir(os.path.join(path, d))]
        info['sessions'] = len(sessions)
        print(f"UE4W: {len(sessions)} сессий")
    
    def _explore_csl(self, path, info):
        files = [f for f in os.listdir(path) if f.endswith('.mat')]
        info['files'] = len(files)
        print(f"CSL: {len(files)} MAT файлов")
    
    def generate_summary(self):
        print(f"\n{'=' * 80}")
        print("SUMMARY")
        print(f"{'=' * 80}")
        
        total_datasets = len(self.report['datasets'])
        accessible = sum(1 for d in self.report['datasets'].values() 
                        if d.get('subjects', d.get('records', d.get('files', 0))) > 0)
        
        all_signals = set()
        for ds in self.report['datasets'].values():
            if 'signals' in ds:
                all_signals.update(ds['signals'])
        
        self.report['summary'] = {
            'total_datasets': total_datasets,
            'accessible_datasets': accessible,
            'all_signals': list(all_signals)[:20]
        }
        
        print(f"📈 Всего датасетов: {total_datasets}")
        print(f"Доступно: {accessible}")
        print(f"📡 Сигналы: {list(all_signals)[:10]}...")
        
        self.report['recommendations'] = [
            "BIDMC: HR, SpO2 (53 субъекта)",
            "WESAD: ECG, EDA, TEMP (15 субъектов)",
            "SPD: HR, IBI, EDA (35 субъектов)",
            "ADARP: HR, EDA (75+ сессий)",
            "💡 Для классификатора: BIDMC + WESAD + SPD",
            "💡 Для GAN: WESAD + WESD + SPD"
        ]
        
        for rec in self.report['recommendations']:
            print(rec)
    
    def save_report(self):
        os.makedirs(OUTPUT_DIR, exist_ok=True)
        
        with open(REPORT_FILE, 'w', encoding='utf-8') as f:
            json.dump(self.report, f, indent=2, ensure_ascii=False, default=str)
        print(f"\n💾 Отчёт: {REPORT_FILE}")
        
        summary_data = []
        for name, info in self.report['datasets'].items():
            summary_data.append({
                'dataset': name,
                'subjects': info.get('subjects', info.get('participants', info.get('records', info.get('files', 0)))),
                'accessible': '✅' if info.get('subjects', info.get('files', 0)) > 0 else '❌'
            })
        
        df = pd.DataFrame(summary_data)
        df.to_csv(f'{OUTPUT_DIR}/dataset_summary.csv', index=False)
        print(f"CSV: {OUTPUT_DIR}/dataset_summary.csv")


def main():
    explorer = DatasetExplorer(RAW_DATA_DIR)
    explorer.explore_all()
    print(f"\n{'=' * 80}")
    print("ГОТОВО!")
    print(f"{'=' * 80}")
    print("\nСледующие шаги:")
    print("1. python preprocess_real_data.py")
    print("2. python train.py")


if __name__ == '__main__':
    main()