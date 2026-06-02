#!/usr/bin/env python3
"""
raw_quality_report.py
Quick diagnostic for your 100 GB raw/ symlinked datasets.
Run this from the project root to see what real signals each dataset actually contains.
"""
import os
import json
from pathlib import Path
from datetime import datetime

RAW_DIR = Path("datasets/raw")
REPORT_FILE = Path("datasets/processed/raw_quality_report.json")

DATASET_EXPECTED = {
    "adarp": {"type": "Empatica E4", "likely_signals": ["HR", "BVP", "EDA"]},
    "wesad": {"type": "Wearable stress", "likely_signals": ["BVP", "EDA", "labels"]},
    "ppg_dalia": {"type": "Field study", "likely_signals": ["BVP", "labels"]},
    "e4selflearning": {"type": "E4 in classes", "likely_signals": ["HR", "BVP"]},
    "big_ideas_lab": {"type": "Glycemic + wearable", "likely_signals": ["HR", "glucose"]},
    "stress_nurses": {"type": "Hospital stress", "likely_signals": ["HR"]},
    "toadstool": {"type": "Gaming stress", "likely_signals": ["HR", "sensor"]},
    "weee": {"type": "Wearable", "likely_signals": ["HR"]},
    "adarp": {"type": "Empatica", "likely_signals": ["HR", "BVP"]},
    "ue4w": {"type": "Unlabeled E4", "likely_signals": ["HR"]},
    "wesd": {"type": "Exam stress", "likely_signals": ["HR"]},
    "bidmc": {"type": "PPG respiration", "likely_signals": ["HR", "RESP"]},
    "spd": {"type": "Stress predict", "likely_signals": ["HR"]},
    "sleep_edf": {"type": "Sleep", "likely_signals": ["EEG", "low HR value"]},
    "capnobase*": {"type": "Respiration benchmark", "likely_signals": ["RESP", "ECG"]},
    "csl": {"type": "Pulse oximetry artifact", "likely_signals": ["SpO2", "artifact labels"]},
}

def scan_dataset(name: str, path: Path) -> dict:
    info = {
        "name": name,
        "path": str(path),
        "exists": path.exists(),
        "size_gb": 0.0,
        "file_count": 0,
        "has_bvp": False,
        "has_eda": False,
        "has_labels": False,
        "has_hr_csv": False,
        "sample_files": [],
        "notes": ""
    }

    if not path.exists():
        return info

    total_size = 0
    count = 0
    for root, dirs, files in os.walk(path):
        for f in files:
            try:
                fp = Path(root) / f
                total_size += fp.stat().st_size
                count += 1
                fname = f.lower()
                if "bvp" in fname or "ppg" in fname:
                    info["has_bvp"] = True
                if "eda" in fname or "gsr" in fname:
                    info["has_eda"] = True
                if "label" in fname or "labels" in fname:
                    info["has_labels"] = True
                if fname == "hr.csv":
                    info["has_hr_csv"] = True
            except:
                pass

    info["size_gb"] = round(total_size / (1024**3), 2)
    info["file_count"] = count

    # Sample top-level files
    try:
        top = list(path.iterdir())[:8]
        info["sample_files"] = [p.name for p in top]
    except:
        pass

    # Heuristic notes
    if info["has_bvp"]:
        info["notes"] = "Good — can compute real HRV"
    elif info["has_hr_csv"]:
        info["notes"] = "Only HR — usable but lower quality"
    elif info["has_labels"]:
        info["notes"] = "Has labels — potentially high value"
    else:
        info["notes"] = "Likely low value for training zones"

    return info

def main():
    print("RAW DATASET QUALITY REPORT")
    print("=" * 70)
    print(f"Scanning: {RAW_DIR.resolve()}")
    print()

    results = []

    for item in sorted(RAW_DIR.iterdir()):
        if not item.is_dir() and not item.is_symlink():
            continue
        name = item.name
        info = scan_dataset(name, item)
        results.append(info)

        print(f"{name:20} | {info['size_gb']:6.2f} GB | "
              f"BVP:{int(info['has_bvp'])} EDA:{int(info['has_eda'])} "
              f"Labels:{int(info['has_labels'])} HR.csv:{int(info['has_hr_csv'])} | {info['notes']}")

    summary = {
        "timestamp": datetime.now().isoformat(),
        "total_size_gb": sum(r["size_gb"] for r in results),
        "datasets": results,
        "recommendations": [
            "Keep (high value): wesad, ppg_dalia, adarp, e4selflearning, stress_nurses, toadstool, weee",
            "Careful (huge): big_ideas_lab (34 GB) — subsample heavily or skip if glucose not needed",
            "Low value / skip: all capnobase_*, csl, sleep_edf (respiration/sleep focused)",
            "Medium: ue4w, wesd, spd, bidmc"
        ]
    }

    REPORT_FILE.parent.mkdir(parents=True, exist_ok=True)
    with open(REPORT_FILE, "w", encoding="utf-8") as f:
        json.dump(summary, f, indent=2, ensure_ascii=False)

    print("\n" + "=" * 70)
    print(f"Report saved to {REPORT_FILE}")
    print(f"Total raw size scanned: {summary['total_size_gb']:.2f} GB")
    print("\nRecommendations:")
    for rec in summary["recommendations"]:
        print(f"  • {rec}")

if __name__ == "__main__":
    main()
