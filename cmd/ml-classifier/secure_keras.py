import io, zipfile, h5py

MAX_ELEMENTS = 10_000_000  # roughly 40MB for float32

def _check_h5_dataset(dataset):
    # Reject extreme shapes
    if any(dim is None or dim > MAX_ELEMENTS for dim in dataset.shape):
        raise ValueError(f"Dataset shape too large: {dataset.shape}")
    # Reject external storage/link
    try:
        external_cnt = dataset.id.get_external_count()
        if external_cnt > 0:
            raise ValueError("Dataset uses external storage/link, which is disallowed")
    except Exception:
        pass

def _inspect_h5_bytes(data: bytes):
    with h5py.File(io.BytesIO(data), "r") as h5:
        def visitor(name, obj):
            if isinstance(obj, h5py.Dataset):
                _check_h5_dataset(obj)
        h5.visititems(visitor)

def secure_load_model(model_path: str):
    """Load a .keras model after validating internal HDF5 files.
    Raises ValueError on unsafe content.
    """
    import keras
    with zipfile.ZipFile(model_path, "r") as z:
        for name in z.namelist():
            if name.endswith('.h5'):
                data = z.read(name)
                _inspect_h5_bytes(data)
    return keras.models.load_model(model_path)
