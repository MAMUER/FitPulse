#!/usr/bin/env python3
"""Prepare classifier dataset from ALL raw physiological datasets."""

from __future__ import annotations

import hashlib
import json
import pickle
import zipfile
from collections import Counter
from pathlib import Path

import numpy as np
import pandas as pd

PROJECT_ROOT = Path(__file__).resolve().parent.parent
RAW_ROOT = PROJECT_ROOT / "datasets" / "raw"
PROCESSED_ROOT = PROJECT_ROOT / "datasets" / "processed"
OUTPUT_FILE = PROCESSED_ROOT / "classifier_dataset.csv"

FEATURE_COLUMNS = [
    "heart_rate",
    "hrv",
    "spo2",
    "temperature",
    "systolic_pressure",
    "diastolic_pressure",
    "sleep_hours",
]

HR_CANDIDATE_COLUMNS = ["hr", "heart_rate", "heart rate", "value"]

HEART_RATE_NAMES = ["heart_rate", "heart rate"]

HR_CSV = "HR.csv"
TEMP_CSV = "TEMP.csv"
IBI_CSV = "IBI.csv"


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------
def md5(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def safe_read_csv(path: Path, **kwargs) -> pd.DataFrame | None:
    try:
        if not path.exists() or path.stat().st_size == 0:
            return None
        return pd.read_csv(path, **kwargs)
    except Exception:
        return None


def norm_name(name: str) -> str:
    return " ".join(name.split()).lower()


def first_existing(df: pd.DataFrame | None, candidates: list[str]) -> str | None:
    if df is None or df.empty:
        return None
    norm_map = {norm_name(c): c for c in df.columns}
    for cand in candidates:
        key = norm_name(cand)
        if key in norm_map:
            return norm_map[key]
    return None


def to_float(val) -> float | None:
    try:
        f = float(val)
        if np.isfinite(f):
            return f
    except (TypeError, ValueError):
        pass
    return None


def read_signal_values(
    path: Path, candidates: list[str], max_rows: int | None = None
) -> pd.Series | None:
    """Read a CSV and return a Series of numeric values for the best matching column."""
    df = safe_read_csv(path, nrows=max_rows)
    if df is None or df.empty:
        return None

    col = first_existing(df, candidates)
    if col is not None:
        return pd.to_numeric(df[col], errors="coerce")

    col = first_existing(df, [c.replace("_", " ") for c in candidates])
    if col is not None:
        return pd.to_numeric(df[col], errors="coerce")

    try:
        first_col_name = str(df.columns[0])
        float(first_col_name)
        if float(first_col_name) > 1e9:
            df2 = safe_read_csv(path, nrows=max_rows, header=None, skiprows=2)
            if df2 is not None and not df2.empty:
                return pd.to_numeric(df2.iloc[:, 0], errors="coerce")
    except Exception:
        pass

    try:
        return pd.to_numeric(df.iloc[:, 0], errors="coerce")
    except Exception:
        pass
    return None


def read_ibi_series(path: Path, max_rows: int | None = None) -> pd.Series | None:
    """Read IBI.csv and return IBI values in seconds."""
    df = safe_read_csv(path, nrows=max_rows)
    if df is None or df.empty:
        return None

    col = first_existing(df, ["ibi", "ibi_value", "ibi", "value"])
    if col is not None:
        return pd.to_numeric(df[col], errors="coerce")

    for candidate_col in df.columns:
        vals = pd.to_numeric(df[candidate_col], errors="coerce").dropna()
        if not vals.empty and len(vals) > 1 and 0.2 < vals.mean() < 2.0:
            return vals
    return None


# ------------------------------------------------------------------
# Labeling
# ------------------------------------------------------------------
def rule_based_label(row: pd.Series) -> str:
    hr = float(row.get("heart_rate", 0) or 0)
    hrv = float(row.get("hrv", 0) or 0)
    spo2 = float(row.get("spo2", 100) or 100)
    temp = float(row.get("temperature", 37) or 37)
    sleep = float(row.get("sleep_hours", 7) or 7)

    labels = [
        ("illness", temp >= 38.0 or spo2 <= 93.0),
        ("recovery", hr < 60 and hrv > 70 and sleep >= 7),
        ("overtraining", hrv < 25 or sleep < 5),
        ("power_hiit", hr >= 160 and hrv >= 45),
        ("endurance_threshold", hr >= 140 and hrv >= 35),
        ("endurance_basic", hr >= 110),
    ]
    for label, condition in labels:
        if condition:
            return label
    return "unknown"


def label_wesad_stress(unique_labels: np.ndarray) -> str:
    counts = Counter(int(x) for x in unique_labels)
    if not counts:
        return "unknown"
    dominant_label = counts.most_common(1)[0][0]
    label_map = {
        0: "recovery",
        1: "overtraining",
        2: "endurance_basic",
        3: "recovery",
        4: "illness",
        6: "overtraining",
        7: "illness",
    }
    return label_map.get(dominant_label, "unknown")


# ------------------------------------------------------------------
# BIDMC
# ------------------------------------------------------------------
def _bidmc_record(
    row: pd.Series, hr_col: str | None, spo2_col: str | None
) -> dict[str, float | None]:
    return {
        "heart_rate": to_float(row.get(hr_col)) if hr_col is not None else None,
        "hrv": None,
        "spo2": to_float(row.get(spo2_col)) if spo2_col is not None else None,
        "temperature": None,
        "systolic_pressure": None,
        "diastolic_pressure": None,
        "sleep_hours": None,
    }


def process_bidmc() -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "bidmc-ppg-and-respiration-dataset-1.0.0"
    if not dataset_dir.exists():
        print("BIDMC: missing")
        return pd.DataFrame()

    records: list[dict[str, float | None]] = []
    files_used = 0
    for subject_idx in range(1, 54):
        subject = f"{subject_idx:02d}"
        numerics_path = dataset_dir / f"bidmc_{subject}_Numerics.csv"
        df = safe_read_csv(numerics_path, low_memory=False)
        if df is None or df.empty:
            continue
        files_used += 1

        hr_col = first_existing(df, ["HR", "Heart Rate", "heart_rate", "PULSE"])
        spo2_col = first_existing(df, ["SpO2", "SpO2 ", "SpO2"])

        if hr_col is None and spo2_col is None:
            print(f"BIDMC {subject}: no usable columns; cols={df.columns.tolist()}")
            continue

        for _, row in df.iterrows():
            records.append(_bidmc_record(row, hr_col, spo2_col))
    print(f"BIDMC files used: {files_used}, rows: {len(records)}")
    return pd.DataFrame(records)


# ------------------------------------------------------------------
# Generic E4-style session processor — returns per-row DataFrame
# ------------------------------------------------------------------
def _find_e4_files(session_dir: Path) -> tuple[Path | None, Path | None, Path | None]:
    hr_path = temp_path = ibi_path = None
    for p in session_dir.iterdir():
        name = p.name.lower()
        if name.startswith("hr") and p.suffix == ".csv":
            hr_path = p
        elif name.startswith("temp") and p.suffix == ".csv":
            temp_path = p
        elif name.startswith("ibi") and p.suffix == ".csv":
            ibi_path = p
    return hr_path, temp_path, ibi_path


def _process_ibi_series(
    ibi_series: pd.Series, index: pd.Index | None
) -> tuple[pd.Series | None, pd.Series | None]:
    if ibi_series is None:
        return None, None
    ibi_vals = ibi_series.reindex(index)
    if ibi_vals is None:
        return None, None
    ibi_arr = ibi_vals.values
    if len(ibi_arr) <= 1:
        return None, None
    ibi_series_for_hr = pd.Series(ibi_arr)
    hr_from_ibi = (
        60.0
        / ibi_series_for_hr.rolling(window=2, min_periods=2)
        .apply(lambda x: 60.0 / np.mean(x) if np.mean(x) > 0 else np.nan, raw=False)
        .to_numpy()
    )
    hrv = pd.Series(ibi_arr).diff().abs() * 1000.0
    return hrv, pd.Series(hr_from_ibi)


def _build_e4_data(hr_series, temp_series, ibi_series, index):
    data = {}
    if hr_series is not None:
        data["heart_rate"] = hr_series.reindex(index)
    if temp_series is not None:
        data["temperature"] = temp_series.reindex(index)
    if ibi_series is not None:
        hrv, hr_from_ibi = _process_ibi_series(ibi_series, index)
        if hrv is not None:
            data["hrv"] = hrv
        if hr_from_ibi is not None:
            data["heart_rate"] = data.get(
                "heart_rate", pd.Series(index=ibi_series.index, dtype=float)
            ).fillna(hr_from_ibi)
    return data


def process_e4_session_rows(
    session_dir: Path, max_rows: int | None = None
) -> pd.DataFrame:
    hr_path, temp_path, ibi_path = _find_e4_files(session_dir)

    hr_series = (
        read_signal_values(hr_path, HR_CANDIDATE_COLUMNS, max_rows=max_rows)
        if hr_path
        else None
    )
    temp_series = (
        read_signal_values(
            temp_path, ["temp", "temperature", "value"], max_rows=max_rows
        )
        if temp_path
        else None
    )
    ibi_series = read_ibi_series(ibi_path, max_rows=max_rows) if ibi_path else None

    if hr_series is None and temp_series is None and ibi_series is None:
        return pd.DataFrame()

    index = next(
        (s.index for s in (hr_series, temp_series, ibi_series) if s is not None), None
    )
    if index is None:
        return pd.DataFrame()

    data = _build_e4_data(hr_series, temp_series, ibi_series, index)
    df = pd.DataFrame(data)
    existing_cols = [c for c in ["heart_rate", "temperature", "hrv"] if c in df.columns]
    if existing_cols:
        df = df.dropna(subset=existing_cols, how="all").reset_index(drop=True)
    return df


def _has_e4_files(session_dir: Path) -> bool:
    return any(
        f.name.lower().startswith(("hr", "temp", "ibi")) and f.suffix == ".csv"
        for f in session_dir.iterdir()
    )


def process_e4_like_dir(
    dataset_dir: Path, max_sessions: int = 200, max_rows_per_file: int | None = None
) -> pd.DataFrame:
    if not dataset_dir.exists():
        print(f"{dataset_dir.name}: missing")
        return pd.DataFrame()

    records: list[pd.DataFrame] = []
    count = 0
    for session_dir in dataset_dir.rglob("*"):
        if not session_dir.is_dir() or not _has_e4_files(session_dir):
            continue

        df = process_e4_session_rows(session_dir, max_rows=max_rows_per_file)
        if not df.empty:
            records.append(df)
        count += 1
        if count >= max_sessions:
            break
    print(f"{dataset_dir.name}: sessions={count}, rows={sum(len(r) for r in records)}")
    if records:
        return pd.concat(records, ignore_index=True)
    return pd.DataFrame()


# ------------------------------------------------------------------
# WESAD
# ------------------------------------------------------------------
def _extract_hr_from_bvp(bvp, fs: float = 32.0) -> tuple[float | None, float | None]:
    if bvp is None or not hasattr(bvp, "mean"):
        return None, None
    bvp_arr = np.asarray(bvp).flatten()
    if len(bvp_arr) == 0:
        return None, None
    try:
        from scipy.signal import find_peaks

        bvp_detrend = bvp_arr - np.mean(bvp_arr)
        peaks, _ = find_peaks(
            bvp_detrend, height=np.std(bvp_detrend) * 0.8, distance=int(fs * 0.3)
        )
        if len(peaks) <= 1:
            return None, None
        rr = np.diff(peaks) / fs
        hr = 60.0 / np.mean(rr)
        hr_val = float(hr) if 20 < hr < 220 else None
        sdnn = np.std(rr) * 1000.0
        hrv_val = float(sdnn) if 5 < sdnn < 500 else None
        return hr_val, hrv_val
    except Exception:
        return None, None


def _extract_hr_from_ecg(ecg, fs: float = 700.0) -> float | None:
    hr_val = None
    if ecg is not None and hasattr(ecg, "mean"):
        ecg_arr = np.asarray(ecg).flatten()
        if len(ecg_arr) > 1000:
            try:
                from scipy.signal import find_peaks

                ecg_detrend = ecg_arr - np.mean(ecg_arr)
                peaks, _ = find_peaks(
                    ecg_detrend,
                    height=np.std(ecg_detrend) * 0.8,
                    distance=int(fs * 0.3),
                )
                if len(peaks) > 1:
                    rr = np.diff(peaks) / fs
                    hr = 60.0 / np.mean(rr)
                    if 20 < hr < 220:
                        hr_val = float(hr)
            except Exception:
                pass
    return hr_val


def process_wesad() -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "WESAD"
    if not dataset_dir.exists():
        print("WESAD: missing")
        return pd.DataFrame()

    records: list[dict[str, float | None]] = []
    for pkl_file in dataset_dir.glob("S*.pkl"):
        try:
            with pkl_file.open("rb") as f:
                data = pickle.load(f, encoding="latin-1")
        except Exception as e:
            print(f"WESAD {pkl_file.name}: pickle failed: {e}")
            continue

        wrist = data.get("signal", {}).get("wrist", {})
        chest = data.get("signal", {}).get("chest", {})
        if not wrist and not chest:
            continue

        temp = wrist.get("TEMP")
        temp_val = (
            to_float(np.nanmean(temp))
            if temp is not None and hasattr(temp, "mean")
            else None
        )

        bvp = wrist.get("BVP")
        hr_val, hrv_val = _extract_hr_from_bvp(bvp, fs=32.0)

        ecg = chest.get("ECG")
        if hr_val is None:
            hr_val = _extract_hr_from_ecg(ecg, fs=700.0)

        eda = chest.get("EDA")
        if eda is not None and hasattr(eda, "mean") and hrv_val is None:
            hrv_val = to_float(np.nanmean(eda))

        records.append(
            {
                "heart_rate": hr_val,
                "hrv": hrv_val,
                "spo2": None,
                "temperature": temp_val,
                "systolic_pressure": None,
                "diastolic_pressure": None,
                "sleep_hours": None,
            }
        )
    print(f"WESAD: subjects={len(records)}")
    return pd.DataFrame(records)


# ------------------------------------------------------------------
# WESAD labels
# ------------------------------------------------------------------
def _wesad_label_record(data: dict) -> dict[str, float | None | str] | None:
    wrist = data.get("signal", {}).get("wrist", {})
    labels = data.get("label")
    if labels is None:
        return None

    temp = wrist.get("TEMP")
    temp_val = (
        to_float(np.nanmean(temp))
        if temp is not None and hasattr(temp, "mean")
        else None
    )

    bvp = wrist.get("BVP")
    hr_val = None
    if bvp is not None and hasattr(bvp, "mean"):
        hr_val = _extract_hr_from_bvp(bvp, fs=32.0)[0]

    label = label_wesad_stress(np.asarray(labels).flatten())
    return {
        "heart_rate": hr_val,
        "hrv": None,
        "spo2": None,
        "temperature": temp_val,
        "systolic_pressure": None,
        "diastolic_pressure": None,
        "sleep_hours": None,
        "_label_override": label,
    }


def process_wesad_labels(max_subjects: int = 15) -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "WESAD"
    if not dataset_dir.exists():
        print("WESAD_labels: missing")
        return pd.DataFrame()

    records: list[dict[str, float | None]] = []
    for pkl_file in sorted(dataset_dir.glob("S*.pkl"))[:max_subjects]:
        try:
            with pkl_file.open("rb") as f:
                data = pickle.load(f, encoding="latin-1")
        except Exception as e:
            print(f"WESAD_labels {pkl_file.name}: pickle failed: {e}")
            continue

        record = _wesad_label_record(data)
        if record is not None:
            records.append(record)
    print(f"WESAD_labels: subjects={len(records)}")
    return pd.DataFrame(records)


# ------------------------------------------------------------------
# WESD zipped exam stress
# ------------------------------------------------------------------
def _extract_wesd_zip(dataset_dir: Path) -> Path | None:
    data_zip = dataset_dir / "Data.zip"
    if not data_zip.exists():
        print("WESD: Data.zip missing")
        return None
    extract_dir = dataset_dir / "extracted"
    if not extract_dir.exists():
        try:
            with zipfile.ZipFile(data_zip, "r") as zf:
                zf.extractall(extract_dir)
        except Exception as e:
            print(f"WESD: extract failed: {e}")
            return None
    return extract_dir


def _process_wesd_participant(
    participant_dir: Path, max_rows_per_file: int | None = None
) -> pd.DataFrame | None:
    hr_files = sorted(participant_dir.glob(HR_CSV))
    temp_files = sorted(participant_dir.glob(TEMP_CSV))
    ibi_files = sorted(participant_dir.glob(IBI_CSV))
    if not hr_files and not temp_files and not ibi_files:
        return None
    df = process_e4_session_rows(participant_dir, max_rows=max_rows_per_file)
    return df if not df.empty else None


def _iter_wesd_participants(
    search_roots: list[Path],
    max_participants: int,
    max_rows_per_file: int | None = None,
) -> tuple[list[pd.DataFrame], int]:
    records: list[pd.DataFrame] = []
    count = 0
    for root in search_roots:
        for participant_dir in root.rglob("*"):
            if not participant_dir.is_dir():
                continue
            df = _process_wesd_participant(participant_dir, max_rows_per_file)
            if df is not None:
                records.append(df)
            count += 1
            if count >= max_participants:
                break
        if count >= max_participants:
            break
    return records, count


def process_wesd(
    max_participants: int = 50, max_rows_per_file: int | None = None
) -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "WESD"
    if not dataset_dir.exists():
        print("WESD: missing")
        return pd.DataFrame()

    extract_dir = _extract_wesd_zip(dataset_dir)
    if extract_dir is None:
        return pd.DataFrame()

    search_roots = [extract_dir]
    raw_dir = RAW_ROOT / "WESD"
    if raw_dir.exists():
        search_roots.append(raw_dir)
    records, count = _iter_wesd_participants(
        search_roots, max_participants, max_rows_per_file
    )
    print(f"WESD: participants={count}, rows={sum(len(r) for r in records)}")
    if records:
        return pd.concat(records, ignore_index=True)
    return pd.DataFrame()


# ------------------------------------------------------------------
# Stress nurses hospital zipped E4
# ------------------------------------------------------------------
def _find_stress_nurse_sessions(participant_dir: Path) -> list[Path]:
    session_dirs: list[Path] = []
    extracted = participant_dir / "extracted"
    if extracted.exists() and extracted.is_dir():
        session_dirs.append(extracted)
    for d in sorted(participant_dir.glob("*_extracted")):
        if d.is_dir():
            session_dirs.append(d)
    return session_dirs


def _extract_stress_nurse_zips(participant_dir: Path) -> list[Path]:
    session_dirs: list[Path] = []
    zips = sorted(participant_dir.glob("*.zip"))
    if zips:
        extract_dir = participant_dir / "extracted"
        try:
            extract_dir.mkdir(exist_ok=True)
            for zf_path in zips:
                try:
                    with zipfile.ZipFile(zf_path, "r") as zf:
                        zf.extractall(extract_dir)
                except Exception:
                    pass
            session_dirs.append(extract_dir)
        except Exception:
            pass
    return session_dirs


def _process_stress_nurse_session(
    session_dir: Path, max_rows_per_file: int | None = None
) -> pd.DataFrame | None:
    hr_files = sorted(session_dir.glob(HR_CSV))
    temp_files = sorted(session_dir.glob(TEMP_CSV))
    ibi_files = sorted(session_dir.glob(IBI_CSV))
    if not hr_files and not temp_files and not ibi_files:
        return None
    df = process_e4_session_rows(session_dir, max_rows=max_rows_per_file)
    return df if not df.empty else None


def process_stress_nurses(
    max_participants: int = 50, max_rows_per_file: int | None = None
) -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "stress_detection_nurses_hospital"
    if not dataset_dir.exists():
        print("stress_nurses: missing")
        return pd.DataFrame()

    records: list[pd.DataFrame] = []
    count = 0
    for participant_dir in dataset_dir.iterdir():
        if not participant_dir.is_dir():
            continue

        session_dirs = _find_stress_nurse_sessions(participant_dir)
        if not session_dirs:
            session_dirs = _extract_stress_nurse_zips(participant_dir)

        for session_dir in session_dirs:
            df = _process_stress_nurse_session(session_dir, max_rows_per_file)
            if df is not None:
                records.append(df)
        count += 1
        if count >= max_participants:
            break
    print(f"stress_nurses: participants={count}, rows={sum(len(r) for r in records)}")
    if records:
        return pd.concat(records, ignore_index=True)
    return pd.DataFrame()


# ------------------------------------------------------------------
# in-gauge_en-gage
# ------------------------------------------------------------------
def _iter_in_gauge_sessions(
    participant_dir: Path, max_rows_per_file: int | None = None
) -> list[pd.DataFrame]:
    records: list[pd.DataFrame] = []
    for session_dir in participant_dir.iterdir():
        if not session_dir.is_dir():
            continue
        df = process_e4_session_rows(session_dir, max_rows=max_rows_per_file)
        if not df.empty:
            records.append(df)
    return records


def process_in_gauge_en_gage(
    max_participants: int = 200, max_rows_per_file: int | None = None
) -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "in-gauge_en-gage"
    if not dataset_dir.exists():
        print("in-gauge_en-gage: missing")
        return pd.DataFrame()

    records: list[pd.DataFrame] = []
    count = 0
    for participant_dir in dataset_dir.iterdir():
        if not participant_dir.is_dir():
            continue
        session_records = _iter_in_gauge_sessions(participant_dir, max_rows_per_file)
        records.extend(session_records)
        count += len(session_records)
        if count >= max_participants:
            break
    print(f"in-gauge_en-gage: sessions={count}, rows={sum(len(r) for r in records)}")
    if records:
        return pd.concat(records, ignore_index=True)
    return pd.DataFrame()


# ------------------------------------------------------------------
# PPG_DaLiA
# ------------------------------------------------------------------
def _process_ppg_dalia_pkl(data: dict) -> dict[str, float | None] | None:
    base_row: dict[str, float | None] = {
        "heart_rate": None,
        "hrv": None,
        "spo2": None,
        "temperature": None,
        "systolic_pressure": None,
        "diastolic_pressure": None,
        "sleep_hours": None,
    }

    rpeaks = data.get("rpeaks")
    if rpeaks is not None and len(rpeaks) > 1:
        rpeaks = np.asarray(rpeaks).flatten()
        diffs = np.diff(rpeaks)
        if np.all(diffs > 0):
            fs = 700.0
            ibi_sec = diffs / fs
            hr = 60.0 / np.mean(ibi_sec)
            if 20 < hr < 220:
                base_row["heart_rate"] = float(hr)
            sdnn_ms = float(np.std(ibi_sec)) * 1000.0
            if 5 < sdnn_ms < 500:
                base_row["hrv"] = sdnn_ms

    wrist = data.get("signal", {}).get("wrist", {})
    temp = wrist.get("TEMP")
    if temp is not None and hasattr(temp, "mean"):
        base_row["temperature"] = to_float(np.nanmean(temp))

    return base_row


def _add_ppg_dalia_hr(records, hr_files):
    if not hr_files:
        return
    hr_series = read_signal_values(
        hr_files[0], ["HR"] + HEART_RATE_NAMES, max_rows=20000
    )
    if hr_series is None or hr_series.dropna().empty:
        return
    for hr_val in hr_series.dropna():
        if 20 <= hr_val <= 220:
            records.append(
                {
                    "heart_rate": float(hr_val),
                    "hrv": None,
                    "spo2": None,
                    "temperature": None,
                    "systolic_pressure": None,
                    "diastolic_pressure": None,
                    "sleep_hours": None,
                }
            )


def _add_ppg_dalia_temp(records, temp_files):
    if not temp_files:
        return
    temp_series = read_signal_values(
        temp_files[0], ["TEMP", "temperature", "temp"], max_rows=20000
    )
    if temp_series is None or temp_series.dropna().empty:
        return
    temp_mean = float(temp_series.dropna().mean())
    records.append(
        {
            "heart_rate": None,
            "hrv": None,
            "spo2": None,
            "temperature": temp_mean,
            "systolic_pressure": None,
            "diastolic_pressure": None,
            "sleep_hours": None,
        }
    )


def _add_ppg_dalia_ibi(records, ibi_files):
    if not ibi_files:
        return
    ibi_series = read_ibi_series(ibi_files[0], max_rows=20000)
    if ibi_series is None or ibi_series.dropna().empty:
        return
    ibi_vals = ibi_series.dropna().values
    if len(ibi_vals) <= 1:
        return
    sdnn_ms = float(np.std(ibi_vals)) * 1000.0
    if not (5 < sdnn_ms < 500):
        return
    records.append(
        {
            "heart_rate": None,
            "hrv": sdnn_ms,
            "spo2": None,
            "temperature": None,
            "systolic_pressure": None,
            "diastolic_pressure": None,
            "sleep_hours": None,
        }
    )


def _add_ppg_dalia_bvp(records, bvp_files):
    if not bvp_files:
        return
    bvp_series = read_signal_values(bvp_files[0], ["BVP", "bvp", "ppg"], max_rows=20000)
    if bvp_series is None or bvp_series.dropna().empty:
        return
    bvp_arr = bvp_series.dropna().values.flatten()
    if len(bvp_arr) <= 1000:
        return
    fs = 64.0
    try:
        from scipy.signal import find_peaks

        bvp_detrend = bvp_arr - np.mean(bvp_arr)
        peaks, _ = find_peaks(
            bvp_detrend, height=np.std(bvp_detrend) * 0.8, distance=int(fs * 0.3)
        )
        if len(peaks) <= 1:
            return
        rr = np.diff(peaks) / fs
        hr = 60.0 / np.mean(rr)
        if not (20 < hr < 220):
            return
        records.append(
            {
                "heart_rate": float(hr),
                "hrv": None,
                "spo2": None,
                "temperature": None,
                "systolic_pressure": None,
                "diastolic_pressure": None,
                "sleep_hours": None,
            }
        )
    except Exception:
        pass


def _process_ppg_dalia_e4(e4_dir: Path) -> list[dict[str, float | None]]:
    records: list[dict[str, float | None]] = []
    hr_files = sorted(e4_dir.glob(HR_CSV))
    temp_files = sorted(e4_dir.glob(TEMP_CSV))
    ibi_files = sorted(e4_dir.glob(IBI_CSV))
    bvp_files = sorted(e4_dir.glob("BVP.csv"))

    _add_ppg_dalia_hr(records, hr_files)
    _add_ppg_dalia_temp(records, temp_files)
    _add_ppg_dalia_ibi(records, ibi_files)
    _add_ppg_dalia_bvp(records, bvp_files)
    return records


def process_ppg_dalia(max_subjects: int = 15) -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "PPG_DaLiA"
    if not dataset_dir.exists():
        print("PPG_DaLiA: missing")
        return pd.DataFrame()

    records: list[dict[str, float | None]] = []
    for subject_dir in sorted(dataset_dir.iterdir())[:max_subjects]:
        if not subject_dir.is_dir():
            continue
        pkl = subject_dir / f"{subject_dir.name}.pkl"
        if pkl.exists():
            try:
                with pkl.open("rb") as f:
                    data = pickle.load(f, encoding="latin-1")
            except Exception as e:
                print(f"PPG_DaLiA {subject_dir.name}: pickle failed: {e}")
                data = {}

            row = _process_ppg_dalia_pkl(data)
            if row is not None:
                records.append(row)

        e4_dir = subject_dir / f"{subject_dir.name}_E4_extracted"
        if e4_dir.exists():
            records.extend(_process_ppg_dalia_e4(e4_dir))

    print(f"PPG_DaLiA: subjects={len(records)}")
    return pd.DataFrame(records)


# ------------------------------------------------------------------
# WEEE (E4 + VO2 + EARBUDS + Fitbit/Apple watch extras)
# ------------------------------------------------------------------
def _process_weee_e4(
    participant_dir: Path, max_rows_per_file: int | None = None
) -> pd.DataFrame | None:
    e4_dir = participant_dir / "E4"
    if not e4_dir.exists():
        return None
    df = process_e4_session_rows(e4_dir, max_rows=max_rows_per_file)
    return df if not df.empty else None


def _process_weee_vo2(
    participant_dir: Path, max_rows_per_file: int | None = None
) -> pd.DataFrame | None:
    vo2_hr = participant_dir / "VO2" / "HeartRateMonitor-Data.csv"
    if not vo2_hr.exists():
        return None
    hr_series = read_signal_values(
        vo2_hr, ["HR[bpm]", "HR"] + HEART_RATE_NAMES, max_rows=max_rows_per_file
    )
    if hr_series is None or hr_series.dropna().empty:
        return None
    return pd.DataFrame({"heart_rate": hr_series.dropna().reset_index(drop=True)})


def _process_weee_earbuds(
    participant_dir: Path, max_rows_per_file: int | None = None
) -> pd.DataFrame | None:
    earbuds = participant_dir / "EARBUDS"
    if not earbuds.exists():
        return None
    for p in earbuds.iterdir():
        if "ppg" in p.name.lower() and p.suffix == ".csv":
            hr_series = read_signal_values(
                p, ["PPG_HR", "hr"] + HEART_RATE_NAMES, max_rows=max_rows_per_file
            )
            if hr_series is not None and not hr_series.dropna().empty:
                return pd.DataFrame(
                    {"heart_rate": hr_series.dropna().reset_index(drop=True)}
                )
            break
    return None


def _process_weee_fitbit_pa(participant_dir: Path) -> pd.DataFrame | None:
    fitbit_pa = participant_dir / "Fitbit" / "Physical Activity"
    if not fitbit_pa.exists():
        return None
    bpm_vals: list[float] = []
    for hr_json in sorted(fitbit_pa.glob("heart_rate-*.json")):
        try:
            with hr_json.open("r", encoding="utf-8") as f:
                data = json.load(f)
            for entry in data:
                val = entry.get("value", {})
                bpm = val.get("bpm")
                if bpm is not None:
                    bpm_vals.append(float(bpm))
        except Exception:
            pass
    if bpm_vals:
        return pd.DataFrame({"heart_rate": bpm_vals})
    return None


def _process_weee_fitbit_other(participant_dir: Path) -> pd.DataFrame | None:
    fitbit_other = participant_dir / "Fitbit" / "Other"
    if not fitbit_other.exists():
        return None
    for eov_csv in sorted(fitbit_other.glob("estimated_oxygen_variation-*.csv")):
        try:
            df = pd.read_csv(eov_csv)
            col = "Infrared to Red Signal Ratio"
            if col in df.columns:
                ratio = pd.to_numeric(df[col], errors="coerce").dropna()
                if not ratio.empty:
                    spo2 = 98.0 - ratio * 0.2
                    spo2 = spo2.clip(70.0, 100.0)
                    return pd.DataFrame({"spo2": spo2.reset_index(drop=True)})
        except Exception:
            pass
    return None


def _parse_weee_metric(metrics: list[dict], name: str) -> list[float]:
    vals: list[float] = []
    for m in metrics:
        if m.get("name") != name:
            continue
        for d in m.get("data", []):
            qty = d.get("qty")
            if qty is not None:
                vals.append(float(qty))
    return vals


def _process_weee_apple(
    participant_dir: Path,
) -> tuple[pd.DataFrame | None, pd.DataFrame | None]:
    apple_dir = participant_dir / "Apple watch"
    if not apple_dir.exists():
        return None, None
    aw_json = apple_dir / "HealthAutoExport.json"
    if not aw_json.exists():
        return None, None
    try:
        with aw_json.open("r", encoding="utf-8") as f:
            data = json.load(f)
        metrics = data.get("data", {}).get("metrics", [])
        hr_vals = _parse_weee_metric(metrics, "heart_rate")
        hrv_vals = _parse_weee_metric(metrics, "heart_rate_variability")
        hr_df = pd.DataFrame({"heart_rate": hr_vals}) if hr_vals else None
        hrv_df = pd.DataFrame({"hrv": hrv_vals}) if hrv_vals else None
        return hr_df, hrv_df
    except Exception:
        return None, None


def _process_weee_zephyr_summary(zephyr_dir: Path, records: list[pd.DataFrame]) -> bool:
    summary_files = sorted(zephyr_dir.glob("*_Summary.csv"))
    if not summary_files:
        summary_files = sorted(zephyr_dir.glob("Summary.csv"))
    if not summary_files:
        return False
    try:
        df_sum = pd.read_csv(summary_files[0])
        hr_col = first_existing(df_sum, ["HR", "heart_rate", "HeartRate"])
        if hr_col:
            hr_vals = (
                pd.to_numeric(df_sum[hr_col], errors="coerce")
                .replace(0, np.nan)
                .dropna()
            )
            hr_vals = hr_vals[(hr_vals > 20) & (hr_vals < 220)]
            if not hr_vals.empty:
                records.append(pd.DataFrame({"heart_rate": hr_vals.tolist()}))
                return True
    except Exception:
        pass
    return False


def _process_weee_zephyr_rr(zephyr_dir: Path, records: list[pd.DataFrame]) -> None:
    rr_files = sorted(zephyr_dir.glob("*_RR.csv"))
    if not rr_files:
        rr_files = sorted(zephyr_dir.glob("RR.csv"))
    if not rr_files:
        return
    try:
        df_rr = pd.read_csv(rr_files[0])
        rr_col = first_existing(df_rr, ["RtoR", "RR", "rr", "ibi", "ibi_value"])
        if rr_col:
            rr_vals = pd.to_numeric(df_rr[rr_col], errors="coerce").dropna()
            rr_vals = rr_vals[(rr_vals > 200) & (rr_vals < 3000)]
            if not rr_vals.empty:
                rr_sec = rr_vals / 1000.0
                sdnn_ms = float(np.std(rr_sec)) * 1000.0
                if 5 < sdnn_ms < 500:
                    records.append(pd.DataFrame({"hrv": [sdnn_ms]}))
    except Exception:
        pass


def _process_weee_zephyr_ecg(zephyr_dir: Path, records: list[pd.DataFrame]) -> None:
    ecg_files = sorted(zephyr_dir.glob("*_ECG.csv"))
    if not ecg_files:
        ecg_files = sorted(zephyr_dir.glob("ECG.csv"))
    if not ecg_files:
        return
    try:
        ecg_series = read_signal_values(
            ecg_files[0], ["EcgWaveform", "ECG", "ecg"], max_rows=20000
        )
        if ecg_series is None or ecg_series.dropna().empty:
            return
        ecg_arr = ecg_series.dropna().values.flatten()
        if len(ecg_arr) <= 1000:
            return
        fs = 256.0
        try:
            from scipy.signal import find_peaks

            ecg_detrend = ecg_arr - np.mean(ecg_arr)
            peaks, _ = find_peaks(
                ecg_detrend, height=np.std(ecg_detrend) * 0.8, distance=int(fs * 0.3)
            )
            if len(peaks) > 1:
                rr = np.diff(peaks) / fs
                hr = 60.0 / np.mean(rr)
                if 20 < hr < 220:
                    records.append(pd.DataFrame({"heart_rate": [float(hr)]}))
        except Exception:
            pass
    except Exception:
        pass


def _process_weee_zephyr_breathing(
    zephyr_dir: Path, records: list[pd.DataFrame]
) -> None:
    br_files = sorted(zephyr_dir.glob("*_Breathing.csv"))
    if not br_files:
        br_files = sorted(zephyr_dir.glob("Breathing.csv"))
    if not br_files:
        return
    try:
        br_series = read_signal_values(
            br_files[0], ["BreathingWaveform", "Breathing", "breathing"], max_rows=20000
        )
        if br_series is None or br_series.dropna().empty:
            return
        br_mean = float(br_series.dropna().mean())
        records.append(pd.DataFrame({"temperature": [br_mean]}))
    except Exception:
        pass


def _process_weee_zephyr(participant_dir: Path) -> list[pd.DataFrame]:
    records: list[pd.DataFrame] = []
    zephyr_dir = participant_dir / "ZEPHYR"
    if not zephyr_dir.exists():
        return records

    hr_found = _process_weee_zephyr_summary(zephyr_dir, records)
    _process_weee_zephyr_rr(zephyr_dir, records)
    if not hr_found:
        _process_weee_zephyr_ecg(zephyr_dir, records)
    _process_weee_zephyr_breathing(zephyr_dir, records)
    return records


def _collect_weee_participant_records(
    participant_dir: Path, max_rows_per_file: int | None = None
) -> list[pd.DataFrame]:
    records: list[pd.DataFrame] = []
    e4_df = _process_weee_e4(participant_dir, max_rows_per_file)
    if e4_df is not None:
        records.append(e4_df)

    vo2_df = _process_weee_vo2(participant_dir, max_rows_per_file)
    if vo2_df is not None:
        records.append(vo2_df)

    earbuds_df = _process_weee_earbuds(participant_dir, max_rows_per_file)
    if earbuds_df is not None:
        records.append(earbuds_df)

    fitbit_pa_df = _process_weee_fitbit_pa(participant_dir)
    if fitbit_pa_df is not None:
        records.append(fitbit_pa_df)

    fitbit_other_df = _process_weee_fitbit_other(participant_dir)
    if fitbit_other_df is not None:
        records.append(fitbit_other_df)

    hr_df, hrv_df = _process_weee_apple(participant_dir)
    if hr_df is not None:
        records.append(hr_df)
    if hrv_df is not None:
        records.append(hrv_df)

    records.extend(_process_weee_zephyr(participant_dir))
    return records


def process_weee(
    max_sessions: int = 200, max_rows_per_file: int | None = None
) -> pd.DataFrame:
    dataset_dir = RAW_ROOT / "WEEE"
    if not dataset_dir.exists():
        print("WEEE: missing")
        return pd.DataFrame()

    records: list[pd.DataFrame] = []
    count = 0
    for participant_dir in dataset_dir.iterdir():
        if not participant_dir.is_dir() or participant_dir.name in {
            "Demographics.csv",
            "Study_Information.csv",
            "Questionnaires",
        }:
            continue

        records.extend(
            _collect_weee_participant_records(participant_dir, max_rows_per_file)
        )
        count += 1
        if count >= max_sessions:
            break
    print(f"WEEE: sessions={count}, rows={sum(len(r) for r in records)}")
    if records:
        return pd.concat(records, ignore_index=True)
    return pd.DataFrame()


# ------------------------------------------------------------------
# Normalize + label
# ------------------------------------------------------------------
def _normalize_column(df: pd.DataFrame, col: str) -> pd.DataFrame:
    if col not in df.columns:
        df[col] = np.nan
    col_min = df[col].min()
    col_max = df[col].max()
    if col_max > col_min:
        df[col] = (df[col] - col_min) / (col_max - col_min)
    return df


def normalize_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    for col in FEATURE_COLUMNS:
        df = _normalize_column(df, col)
    return df


def apply_label_overrides(df: pd.DataFrame) -> pd.DataFrame:
    if "_label_override" not in df.columns:
        return df
    df["label"] = df.apply(
        lambda row: (
            row["_label_override"]
            if pd.notna(row["_label_override"])
            else rule_based_label(row)
        ),
        axis=1,
    )
    return df.drop(columns=["_label_override"])


# ------------------------------------------------------------------
# Main
# ------------------------------------------------------------------
def main() -> None:
    PROCESSED_ROOT.mkdir(parents=True, exist_ok=True)

    summary = []

    processors = [
        ("bidmc", process_bidmc()),
        (
            "E4SelfLearning_ADARP",
            process_e4_like_dir(RAW_ROOT / "ADARP", max_sessions=9999),
        ),
        (
            "E4SelfLearning_big-ideas",
            process_e4_like_dir(RAW_ROOT / "big-ideas", max_sessions=9999),
        ),
        ("in-gauge_en-gage", process_in_gauge_en_gage(max_participants=9999)),
        ("PPG_DaLiA", process_ppg_dalia(max_subjects=999)),
        ("SPD", process_e4_like_dir(RAW_ROOT / "SPD", max_sessions=9999)),
        ("stress_nurses", process_stress_nurses(max_participants=9999)),
        ("Toadstool", process_e4_like_dir(RAW_ROOT / "Toadstool", max_sessions=9999)),
        ("ue4w", process_e4_like_dir(RAW_ROOT / "ue4w", max_sessions=9999)),
        ("WEEE", process_weee(max_sessions=9999)),
        ("WESAD", process_wesad()),
        ("WESAD_labels", process_wesad_labels(max_subjects=999)),
        ("WESD", process_wesd(max_participants=9999)),
    ]

    for name, df in processors:
        if df is None or df.empty:
            continue
        summary.append(df)

    if not summary:
        raise RuntimeError("No raw datasets processed")

    df = pd.concat(summary, ignore_index=True)
    print(f"Combined rows before cleanup: {len(df)}")

    df = df.dropna(subset=FEATURE_COLUMNS, how="all").reset_index(drop=True)
    df = df.drop_duplicates(subset=FEATURE_COLUMNS, keep="first").reset_index(drop=True)
    print(f"Rows after dropping all-NaN and duplicates: {len(df)}")

    if "_label_override" in df.columns:
        df = apply_label_overrides(df)
    else:
        df["label"] = df.apply(rule_based_label, axis=1)

    df = normalize_features(df)
    df = df.reset_index(drop=True)

    PROCESSED_ROOT.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUTPUT_FILE, index=False)
    print(f"\nDataset saved to {OUTPUT_FILE}")
    print(f"Total samples: {len(df)}")
    print("Class distribution:")
    print(df["label"].value_counts().to_string())
    print("Feature stats:")
    print(df[FEATURE_COLUMNS].describe().to_string())


if __name__ == "__main__":
    main()
