"""
Анализ распределения пульса по всем датасетам
"""
import os
import pandas as pd
import numpy as np
import pickle
import matplotlib.pyplot as plt

RAW_DATA_DIR = '../../datasets/raw'
OUTPUT_DIR = '../../datasets/processed'

def analyze_hr_in_dataset(dataset_name, process_func):
    """Analyze HR distribution in a dataset"""
    print(f"\n{'='*60}")
    print(f"ANALYZING: {dataset_name}")
    print(f"{'='*60}")
    
    all_hr = []
    
    try:
        records = process_func()
        for r in records:
            if 'hr' in r and r['hr'] is not None:
                all_hr.append(r['hr'])
        
        if len(all_hr) > 0:
            hr_array = np.array(all_hr)
            print(f"Total samples: {len(hr_array)}")
            print(f"HR Min: {hr_array.min():.1f} bpm")
            print(f"HR Max: {hr_array.max():.1f} bpm")
            print(f"HR Mean: {hr_array.mean():.1f} bpm")
            print(f"HR Std: {hr_array.std():.1f} bpm")
            
            # HR Zones
            zones = {
                'Recovery (50-65% HRmax)': len(hr_array[(hr_array >= 50) & (hr_array < 100)]),
                'E1-E2 (65-80% HRmax)': len(hr_array[(hr_array >= 100) & (hr_array < 130)]),
                'E3 (80-90% HRmax)': len(hr_array[(hr_array >= 130) & (hr_array < 155)]),
                'HIIT (90-100% HRmax)': len(hr_array[(hr_array >= 155) & (hr_array <= 200)])
            }
            
            print(f"\nHR Zone Distribution:")
            for zone, count in zones.items():
                pct = (count / len(hr_array)) * 100
                print(f"  {zone}: {count} ({pct:.1f}%)")
            
            return hr_array
        else:
            print("No HR data found!")
            return np.array([])
            
    except Exception as e:
        print(f"Error: {e}")
        return np.array([])

def main():
    print("HR DISTRIBUTION ANALYSIS")
    print(f"Raw data directory: {RAW_DATA_DIR}")
    
    # Collect all HR data
    all_hr_combined = []
    
    # Analyze each dataset
    from preprocess_real_data import (
        process_bidmc, process_wesad, process_spd, 
        process_wesd, process_adarp
    )
    
    datasets = [
        ('BIDMC', process_bidmc),
        ('WESAD', process_wesad),
        ('SPD', process_spd),
        ('WESD', process_wesd),
        ('ADARP', process_adarp),
    ]
    
    for name, func in datasets:
        hr_data = analyze_hr_in_dataset(name, func)
        if len(hr_data) > 0:
            all_hr_combined.extend(hr_data.tolist())
    
    # Combined analysis
    if len(all_hr_combined) > 0:
        all_hr = np.array(all_hr_combined)
        
        print(f"\n{'='*60}")
        print("COMBINED ANALYSIS (ALL DATASETS)")
        print(f"{'='*60}")
        print(f"Total samples: {len(all_hr)}")
        print(f"HR Range: {all_hr.min():.1f} - {all_hr.max():.1f} bpm")
        print(f"HR Mean: {all_hr.mean():.1f} bpm")
        
        # Histogram
        plt.figure(figsize=(12, 6))
        plt.hist(all_hr, bins=50, edgecolor='black', alpha=0.7)
        plt.axvline(x=100, color='g', linestyle='--', label='E1-E2 threshold (100 bpm)')
        plt.axvline(x=130, color='y', linestyle='--', label='E3 threshold (130 bpm)')
        plt.axvline(x=155, color='r', linestyle='--', label='HIIT threshold (155 bpm)')
        plt.xlabel('Heart Rate (bpm)')
        plt.ylabel('Frequency')
        plt.title('HR Distribution Across All Datasets')
        plt.legend()
        plt.grid(True, alpha=0.3)
        
        os.makedirs(OUTPUT_DIR, exist_ok=True)
        plt.savefig(f'{OUTPUT_DIR}/hr_distribution.png', dpi=150)
        print(f"\nHistogram saved to: {OUTPUT_DIR}/hr_distribution.png")
        
        # Save stats
        stats = {
            'total_samples': len(all_hr),
            'hr_min': float(all_hr.min()),
            'hr_max': float(all_hr.max()),
            'hr_mean': float(all_hr.mean()),
            'hr_std': float(all_hr.std()),
            'recovery_count': int(len(all_hr[all_hr < 100])),
            'e1e2_count': int(len(all_hr[(all_hr >= 100) & (all_hr < 130)])),
            'e3_count': int(len(all_hr[(all_hr >= 130) & (all_hr < 155)])),
            'hiit_count': int(len(all_hr[all_hr >= 155]))
        }
        
        import json
        with open(f'{OUTPUT_DIR}/hr_analysis.json', 'w') as f:
            json.dump(stats, f, indent=2)
        print(f"Stats saved to: {OUTPUT_DIR}/hr_analysis.json")

if __name__ == '__main__':
    main()