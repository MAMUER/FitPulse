"""Минимальный ридер EDF/EDF+ без внешних зависимостей.

Нужен для гипнограмм Sleep-EDF: mne не читает их как EDF+ с сигналом.
Формат: 256-байтный заголовок + по 256 байт на сигнал, затем записи int16 LE.
"""

from __future__ import annotations

import struct
from pathlib import Path


def _s(raw: bytes, start: int, length: int) -> str:
    return raw[start : start + length].decode("ascii", "ignore").strip()


def read_edf(path: str | Path) -> tuple[list[str], list[list[float]], float]:
    """Возвращает (названия сигналов, данные [сигнал][отсчёт], длительность)."""
    data = Path(path).read_bytes()
    if len(data) < 256:
        raise ValueError("файл слишком мал")

    header_bytes = int(_s(data, 184, 8))
    n_records = int(_s(data, 236, 8))
    duration = float(_s(data, 244, 8))
    n_signals = int(_s(data, 252, 4))

    labels: list[str] = []
    samples_per_record: list[int] = []
    phys_min: list[float] = []
    phys_max: list[float] = []
    dig_min: list[int] = []
    dig_max: list[int] = []

    offset = 256
    for _ in range(n_signals):
        labels.append(_s(data, offset, 16))
        offset += 16 + 80 + 8
        samples_per_record.append(int(_s(data, offset, 8)))
        offset += 8
        phys_min.append(float(_s(data, offset, 8)))
        offset += 8
        phys_max.append(float(_s(data, offset, 8)))
        offset += 8
        dig_min.append(int(_s(data, offset, 8)))
        offset += 8
        dig_max.append(int(_s(data, offset, 8)))
        offset += 8 + 80 + 32

    records: list[list[float]] = [[] for _ in range(n_signals)]
    total_samples = sum(samples_per_record)
    if n_records <= 0:
        n_records = (len(data) - header_bytes) // max(1, total_samples * 2)

    pos = header_bytes
    for _ in range(n_records):
        for sig in range(n_signals):
            count = samples_per_record[sig]
            if count <= 0:
                continue
            raw = data[pos : pos + count * 2]
            pos += count * 2
            if len(raw) < count * 2:
                break
            values = struct.unpack("<%dh" % count, raw)
            scale = 1.0
            if dig_max[sig] != dig_min[sig]:
                scale = (phys_max[sig] - phys_min[sig]) / (dig_max[sig] - dig_min[sig])
            for v in values:
                records[sig].append(phys_min[sig] + (v - dig_min[sig]) * scale)

    return labels, records, duration
def read_edf_annotations(path: str | Path) -> list[tuple[float, list[str]]]:
    """Читает EDF+ аннотации (TAL) — например, гипнограмму Sleep-EDF.

    Формат TAL: `+onset\x15duration\x14text\x14\x00`.
    Возвращает список (onset_секунды, [тексты]).
    """
    data = Path(path).read_bytes()
    if len(data) < 256:
        return []

    header_bytes = int(_s(data, 184, 8))
    n_records = int(_s(data, 236, 8))
    if n_records <= 0:
        n_records = 1

    record_size = (len(data) - header_bytes) // n_records
    if record_size <= 0:
        return []

    annotations: list[tuple[float, list[str]]] = []
    pos = header_bytes
    for _ in range(n_records):
        record = data[pos : pos + record_size]
        pos += record_size

        tokens = record.split(b"\x14")
        i = 0
        while i < len(tokens):
            token = tokens[i].lstrip(b"\x00")
            if b"\x15" not in token:
                i += 1
                continue
            onset_field = token.split(b"\x15", 1)[0].lstrip(b"+")
            try:
                onset = float(onset_field)
            except ValueError:
                i += 1
                continue
            texts: list[str] = []
            if i + 1 < len(tokens):
                text = tokens[i + 1].strip(b"\x00").decode("ascii", "ignore").strip()
                if text:
                    texts.append(text)
            if texts:
                annotations.append((onset, texts))
            i += 2
    return annotations
