"""Минимальный ридер EDF/EDF+ без внешних зависимостей.

Нужен для гипнограмм Sleep-EDF: mne не читает их как EDF+ с сигналом.
Формат: 256-байтный заголовок + по 256 байт на сигнал, затем записи int16 LE.
"""

from __future__ import annotations

import struct
from pathlib import Path


def _s(raw: bytes, start: int, length: int) -> str:
    return raw[start : start + length].decode("ascii", "ignore").strip()


def _read_signal_headers(
    data: bytes, n_signals: int
) -> tuple[list[str], list[int], list[float], list[float], list[int], list[int]]:
    labels: list[str] = []
    samples: list[int] = []
    pmin: list[float] = []
    pmax: list[float] = []
    dmin: list[int] = []
    dmax: list[int] = []

    offset = 256
    for _ in range(n_signals):
        labels.append(_s(data, offset, 16))
        offset += 16 + 80 + 8
        samples.append(int(_s(data, offset, 8)))
        offset += 8
        pmin.append(float(_s(data, offset, 8)))
        offset += 8
        pmax.append(float(_s(data, offset, 8)))
        offset += 8
        dmin.append(int(_s(data, offset, 8)))
        offset += 8
        dmax.append(int(_s(data, offset, 8)))
        offset += 8 + 80 + 32
    return labels, samples, pmin, pmax, dmin, dmax


def _scale(dmin: int, dmax: int, pmin: float, pmax: float) -> float:
    if dmax == dmin:
        return 1.0
    return (pmax - pmin) / (dmax - dmin)


def _read_records(
    data: bytes,
    header_bytes: int,
    n_records: int,
    samples: list[int],
    pmin: list[float],
    pmax: list[float],
    dmin: list[int],
    dmax: list[int],
) -> list[list[float]]:
    records: list[list[float]] = [[] for _ in samples]
    total = sum(samples)
    if n_records <= 0:
        n_records = (len(data) - header_bytes) // max(1, total * 2)

    pos = header_bytes
    for _ in range(n_records):
        for sig, count in enumerate(samples):
            if count <= 0:
                continue
            raw = data[pos : pos + count * 2]
            pos += count * 2
            if len(raw) < count * 2:
                break
            scale = _scale(dmin[sig], dmax[sig], pmin[sig], pmax[sig])
            for value in struct.unpack("<%dh" % count, raw):
                records[sig].append(pmin[sig] + (value - dmin[sig]) * scale)
    return records


def read_edf(path: str | Path) -> tuple[list[str], list[list[float]], float]:
    """Возвращает (названия сигналов, данные [сигнал][отсчёт], длительность)."""
    data = Path(path).read_bytes()
    if len(data) < 256:
        raise ValueError("файл слишком мал")

    header_bytes = int(_s(data, 184, 8))
    n_records = int(_s(data, 236, 8))
    duration = float(_s(data, 244, 8))
    n_signals = int(_s(data, 252, 4))

    labels, samples, pmin, pmax, dmin, dmax = _read_signal_headers(data, n_signals)
    records = _read_records(
        data, header_bytes, n_records, samples, pmin, pmax, dmin, dmax
    )
    return labels, records, duration


def _parse_tal(record: bytes) -> list[tuple[float, list[str]]]:
    """Разбирает TAL-записи EDF+: `+onset\\x15duration\\x14text\\x14\\x00`."""
    annotations: list[tuple[float, list[str]]] = []
    tokens = record.split(b"\x14")
    i = 0
    while i < len(tokens):
        token = tokens[i].lstrip(b"\x00")
        if b"\x15" not in token:
            i += 1
            continue
        try:
            onset = float(token.split(b"\x15", 1)[0].lstrip(b"+"))
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


def read_edf_annotations(path: str | Path) -> list[tuple[float, list[str]]]:
    """Читает EDF+ аннотации (TAL) — например, гипнограмму Sleep-EDF."""
    data = Path(path).read_bytes()
    if len(data) < 256:
        return []

    header_bytes = int(_s(data, 184, 8))
    n_records = int(_s(data, 236, 8)) or 1
    record_size = (len(data) - header_bytes) // n_records
    if record_size <= 0:
        return []

    annotations: list[tuple[float, list[str]]] = []
    pos = header_bytes
    for _ in range(n_records):
        annotations.extend(_parse_tal(data[pos : pos + record_size]))
        pos += record_size
    return annotations
