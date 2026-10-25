"""Tests for gan_train.py and gan_model.py with comprehensive torch mocking.

Torch is not loadable on this system (DLL error), so we inject a fake torch
module into sys.modules before importing gan_train/gan_model.
"""

import contextlib
import os
import sys
import tempfile
import types
import unittest
from pathlib import Path
from unittest.mock import patch

import numpy as np

_rng = np.random.default_rng(seed=42)

MODEL_PT = "model.pt"
DATA_CSV = "data.csv"
GENERATOR_FILENAME = "gan_generator.pt"


def _touch_path(path):
    Path(path).parent.mkdir(parents=True, exist_ok=True)
    Path(path).touch()
    return None


class _Tensor:
    """Minimal torch.Tensor-like wrapper around numpy arrays."""

    def __init__(self, data=None):
        if data is None:
            self._data = np.zeros((0,), dtype=np.float32)
        else:
            self._data = np.asarray(data, dtype=np.float32)

    def __array__(self):
        return self._data

    def to(self, *_a, **_k):
        return self

    def detach(self):
        return self

    def item(self):
        if self._data.size == 1:
            return float(self._data.flat[0])
        return self._data.tolist()

    def size(self, dim=None):
        if dim is None:
            return int(self._data.size)
        return int(self._data.shape[dim])

    def dim(self):
        return int(self._data.ndim)

    def ndim(self):
        return int(self._data.ndim)

    def unsqueeze(self, dim=0):
        return _Tensor(np.expand_dims(self._data, axis=dim))

    def repeat(self, *dims):
        return _Tensor(np.tile(self._data, dims))

    def cpu(self):
        return self

    def numpy(self):
        return self._data

    def __len__(self):
        return int(self._data.shape[0])

    def __getitem__(self, idx):
        return _Tensor(self._data[idx])

    def __add__(self, other):
        if isinstance(other, _Tensor):
            return _Tensor(self._data + other._data)
        return _Tensor(self._data + np.asarray(other))

    def __radd__(self, other):
        return self.__add__(other)

    def backward(self):
        """No-op: fake tensor has no gradient computation."""
        pass

    def __getattr__(self, name):
        if hasattr(self._data, name):
            return getattr(self._data, name)
        raise AttributeError(f"'{type(self).__name__}' has no attribute '{name}'")


# ---------------------------------------------------------------------------
# Inject a fake torch module
# ---------------------------------------------------------------------------
_fake_torch = types.ModuleType("torch")
for _name, _value in [
    ("device", lambda *a, **k: "cpu"),
    ("Tensor", _Tensor),
    ("FloatTensor", lambda *a, **k: _Tensor(np.asarray(a[0] if a else []))),
    (
        "randn",
        lambda *a, **k: _Tensor(
            _rng.standard_normal(size=a).astype(np.float32)  # type: ignore[attr-defined]
        ),
    ),
    ("zeros", lambda *a, **k: _Tensor(np.zeros(a, dtype=np.float32))),
    ("ones", lambda *a, **k: _Tensor(np.ones(a, dtype=np.float32))),
    (
        "cat",
        lambda tensors, dim=0: _Tensor(
            np.concatenate(
                [np.asarray(t._data if isinstance(t, _Tensor) else t) for t in tensors],
                axis=dim,
            )
        ),
    ),
    ("no_grad", contextlib.nullcontext),
    (
        "save",
        lambda obj, path: _touch_path(path),
    ),
    (
        "load",
        lambda *a, **k: {
            "generator": {},
            "discriminator": {},
            "opt_g": {},
            "opt_d": {},
        },
    ),
    ("cuda", types.SimpleNamespace(is_available=lambda: False)),
]:
    setattr(_fake_torch, _name, _value)
sys.modules["torch"] = _fake_torch


class _Module:
    def __init__(self):
        self._modules = {}

    def __setattr__(self, name, value):
        if isinstance(value, _Module):
            self._modules[name] = value
        super().__setattr__(name, value)

    def __call__(self, *a, **k):
        return self.forward(*a, **k)

    def forward(self, *a, **k):
        raise NotImplementedError

    def parameters(self):
        return []

    def eval(self):
        return self

    def train(self):
        return self

    def to(self, *_a, **_k):
        return self

    def state_dict(self):
        return {}

    def load_state_dict(self, state_dict):
        """No-op: fake module has no state to load."""
        pass


class _Sequential(_Module):
    def __init__(self, *args):
        super().__init__()
        for i, m in enumerate(args):
            self._modules[str(i)] = m

    def forward(self, x):
        for m in self._modules.values():
            x = m(x)
        return _Tensor(x) if not isinstance(x, _Tensor) else x


def _make_layer(in_f=None, out_f=None):
    m = _Module()

    def forward(x):
        x_arr = np.asarray(x, dtype=np.float32)
        if in_f is not None and out_f is not None:
            w = _rng.standard_normal((out_f, in_f)).astype(np.float32) * 0.1
            return x_arr @ w.T
        return x_arr

    m.forward = forward
    return m


class _BCELoss(_Module):
    def forward(self, x, y):
        return _Tensor(np.array([0.5]))


_fake_nn = types.ModuleType("torch.nn")
for _name, _value in [
    ("Module", _Module),
    ("Sequential", _Sequential),
    ("Linear", _make_layer),
    ("LeakyReLU", _make_layer),
    ("BatchNorm1d", _make_layer),
    ("Tanh", _make_layer),
    ("Dropout", _make_layer),
    ("Sigmoid", _make_layer),
    ("BCELoss", _BCELoss),
]:
    setattr(_fake_nn, _name, _value)
sys.modules["torch.nn"] = _fake_nn


class _Adam:
    def __init__(self, params, *_a, **_k):
        """No-op: fake optimizer does not track parameters."""
        pass

    def zero_grad(self):
        """No-op: fake optimizer has no gradients to zero."""
        pass

    def step(self):
        """No-op: fake optimizer performs no optimization step."""
        pass

    def state_dict(self):
        return {}

    def load_state_dict(self, state_dict):
        """No-op: fake optimizer has no state to load."""
        pass


_fake_optim = types.ModuleType("torch.optim")
for _name, _value in [("Adam", _Adam)]:
    setattr(_fake_optim, _name, _value)
sys.modules["torch.optim"] = _fake_optim

_fake_utils = types.ModuleType("torch.utils")
_fake_data = types.ModuleType("torch.utils.data")
setattr(
    _fake_data,
    "DataLoader",
    lambda dataset, batch_size=1, shuffle=False, num_workers=0: [
        (
            _Tensor(
                np.stack(
                    [
                        dataset[i][0]._data
                        for i in range(j, min(j + batch_size, len(dataset)))
                    ]
                )
            ),
            _Tensor(
                np.stack(
                    [
                        dataset[i][1]._data
                        for i in range(j, min(j + batch_size, len(dataset)))
                    ]
                )
            ),
        )
        for j in range(0, len(dataset), batch_size)
    ],
)
setattr(_fake_data, "Dataset", object)
sys.modules["torch.utils"] = _fake_utils
sys.modules["torch.utils.data"] = _fake_data

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gan_model import (  # noqa: E402
    PLAN_DIM,
    ConditionalGAN,
    Discriminator,
    Generator,
    TrainingPlanDataset,
    build_condition,
    denormalize_plan,
    normalize_plan,
)
from gan_train import (  # noqa: E402
    _safe_path,
    generate_plan,
    load_dataset,
    parser,
    train,
)


class TestSafePath(unittest.TestCase):
    def setUp(self):
        self.models_dir = Path("models").resolve()
        self.models_dir.mkdir(parents=True, exist_ok=True)

    def tearDown(self):
        import shutil

        if self.models_dir.exists():
            shutil.rmtree(self.models_dir)

    def test_valid_path_inside_models(self):
        p = self.models_dir / GENERATOR_FILENAME
        result = _safe_path(p)
        self.assertEqual(result, p.resolve())

    def test_path_traversal_blocked(self):
        p = Path("outside") / MODEL_PT
        with self.assertRaises(ValueError):
            _safe_path(p)

    def test_traversal_with_dotdot(self):
        p = self.models_dir / ".." / "escape.pt"
        with self.assertRaises(ValueError):
            _safe_path(p)


class TestLoadDataset(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def _write_csv(self, rows):
        path = os.path.join(self.tmpdir, "plans.csv")
        header = (
            "c0,c1,c2,c3,c4,c5,c6,c7,c8,c9,c10,c11,c12,c13,c14,c15,c16,c17,c18,"
            "klass,age,gender,weight,height,goal\n"
        )
        with open(path, "w") as f:
            f.write(header)
            for row in rows:
                f.write(",".join(str(v) for v in row) + "\n")
        return path

    def test_valid_dataset(self):
        plans_row = [float(i) for i in range(PLAN_DIM)]
        row1 = plans_row + [0, 30.0, 1.0, 75.0, 180.0, 1.0]
        row2 = plans_row + [1, 25.0, 0.0, 60.0, 165.0, 2.0]
        path = self._write_csv([row1, row2])
        plans, conditions = load_dataset(path)
        self.assertEqual(plans.shape, (2, PLAN_DIM))
        self.assertEqual(conditions.shape[0], 2)
        self.assertEqual(conditions.shape[1], 12)

    def test_missing_file(self):
        path = os.path.join(self.tmpdir, "nonexistent.csv")
        with self.assertRaises(FileNotFoundError):
            load_dataset(path)

    def test_negative_plan_values(self):
        plans_row = [float(i) for i in range(PLAN_DIM)]
        plans_row[0] = -1.0
        row1 = plans_row + [0, 30.0, 1.0, 75.0, 180.0, 1.0]
        row2 = plans_row + [1, 25.0, 0.0, 60.0, 165.0, 2.0]
        path = self._write_csv([row1, row2])
        with self.assertRaises(ValueError):
            load_dataset(path)

    def test_negative_user_features(self):
        plans_row = [float(i) for i in range(PLAN_DIM)]
        row1 = plans_row + [0, -30.0, 1.0, 75.0, 180.0, 1.0]
        row2 = plans_row + [1, 25.0, 0.0, 60.0, 165.0, 2.0]
        path = self._write_csv([row1, row2])
        with self.assertRaises(ValueError):
            load_dataset(path)


class TestBuildCondition(unittest.TestCase):
    def test_one_hot_encoding(self):
        features = np.array([30.0, 1.0, 75.0, 180.0, 1.0], dtype=np.float32)
        cond = build_condition(2, features)
        self.assertEqual(cond.shape, (12,))
        self.assertEqual(cond[2], 1.0)
        self.assertEqual(np.sum(cond[:7]), 1.0)
        np.testing.assert_array_equal(cond[7:], features)

    def test_class_index_zero(self):
        features = np.array([25.0, 0.0, 60.0, 165.0, 2.0], dtype=np.float32)
        cond = build_condition(0, features)
        self.assertEqual(cond[0], 1.0)
        self.assertEqual(np.sum(cond[:7]), 1.0)

    def test_class_index_max(self):
        features = np.array([40.0, 1.0, 80.0, 190.0, 3.0], dtype=np.float32)
        cond = build_condition(6, features)
        self.assertEqual(cond[6], 1.0)
        self.assertEqual(np.sum(cond[:7]), 1.0)


class TestNormalizePlan(unittest.TestCase):
    def test_normalize_basic(self):
        plan = np.array([0.0, 50.0, 100.0], dtype=np.float32)
        plan_min = np.array([0.0, 0.0, 0.0], dtype=np.float32)
        plan_max = np.array([100.0, 100.0, 100.0], dtype=np.float32)
        result = normalize_plan(plan, plan_min, plan_max)
        np.testing.assert_almost_equal(result[0], -1.0)
        np.testing.assert_almost_equal(result[1], 0.0)
        np.testing.assert_almost_equal(result[2], 1.0)

    def test_normalize_identical_min_max(self):
        plan = np.array([5.0, 5.0], dtype=np.float32)
        plan_min = np.array([5.0, 5.0], dtype=np.float32)
        plan_max = np.array([5.0, 5.0], dtype=np.float32)
        result = normalize_plan(plan, plan_min, plan_max)
        np.testing.assert_almost_equal(result[0], -1.0)
        np.testing.assert_almost_equal(result[1], -1.0)

    def test_normalize_out_of_range(self):
        plan = np.array([200.0], dtype=np.float32)
        plan_min = np.array([0.0], dtype=np.float32)
        plan_max = np.array([100.0], dtype=np.float32)
        result = normalize_plan(plan, plan_min, plan_max)
        self.assertGreater(result[0], 1.0)


class TestDenormalizePlan(unittest.TestCase):
    def test_denormalize_basic(self):
        plan_norm = np.array([-1.0, 0.0, 1.0], dtype=np.float32)
        plan_min = np.array([0.0, 0.0, 0.0], dtype=np.float32)
        plan_max = np.array([100.0, 100.0, 100.0], dtype=np.float32)
        result = denormalize_plan(plan_norm, plan_min, plan_max)
        np.testing.assert_almost_equal(result[0], 0.0)
        np.testing.assert_almost_equal(result[1], 1.0)
        np.testing.assert_almost_equal(result[2], 1.0)

    def test_denormalize_clips_above_one(self):
        plan_norm = np.array([2.0], dtype=np.float32)
        plan_min = np.array([0.0], dtype=np.float32)
        plan_max = np.array([100.0], dtype=np.float32)
        result = denormalize_plan(plan_norm, plan_min, plan_max)
        self.assertEqual(float(result[0]), 1.0)

    def test_denormalize_clips_below_zero(self):
        plan_norm = np.array([-2.0], dtype=np.float32)
        plan_min = np.array([0.0], dtype=np.float32)
        plan_max = np.array([100.0], dtype=np.float32)
        result = denormalize_plan(plan_norm, plan_min, plan_max)
        self.assertEqual(float(result[0]), 0.0)


class TestTrainingPlanDataset(unittest.TestCase):
    def test_dataset_init_and_len(self):
        plans = np.zeros((5, PLAN_DIM), dtype=np.float32)
        conditions = np.zeros((5, 12), dtype=np.float32)
        ds = TrainingPlanDataset(plans, conditions)
        self.assertEqual(len(ds), 5)

    def test_dataset_getitem(self):
        plans = np.ones((3, PLAN_DIM), dtype=np.float32)
        conditions = np.ones((3, 12), dtype=np.float32)
        ds = TrainingPlanDataset(plans, conditions)
        p, c = ds[0]
        self.assertEqual(len(p), PLAN_DIM)
        self.assertEqual(len(c), 12)


class TestGenerator(unittest.TestCase):
    def test_generator_init(self):
        g = Generator()
        self.assertEqual(g.latent_dim, 100)
        self.assertEqual(g.condition_dim, 12)
        self.assertEqual(g.plan_dim, 19)

    def test_generator_forward(self):
        g = Generator()
        z = np.zeros((2, 100), dtype=np.float32)
        cond = np.zeros((2, 12), dtype=np.float32)
        out = g(z, cond)
        self.assertEqual(out.shape, (2, 19))


class TestDiscriminator(unittest.TestCase):
    def test_discriminator_init(self):
        d = Discriminator()
        self.assertIsNotNone(d.model)

    def test_discriminator_forward(self):
        d = Discriminator()
        plan = np.zeros((2, 19), dtype=np.float32)
        cond = np.zeros((2, 12), dtype=np.float32)
        out = d(plan, cond)
        self.assertEqual(out.shape, (2, 1))


class TestConditionalGAN(unittest.TestCase):
    def setUp(self):
        self.gan = ConditionalGAN(device="cpu")

    def test_gan_init(self):
        self.assertIsNotNone(self.gan.generator)
        self.assertIsNotNone(self.gan.discriminator)
        self.assertEqual(self.gan.latent_dim, 100)
        self.assertEqual(self.gan.condition_dim, 12)

    def test_gan_train_step(self):
        batch_size = 4
        real_plans = _Tensor(np.zeros((batch_size, PLAN_DIM), dtype=np.float32))
        real_conditions = _Tensor(np.zeros((batch_size, 12), dtype=np.float32))
        metrics = self.gan.train_step(real_plans, real_conditions)
        self.assertIn("d_loss", metrics)
        self.assertIn("g_loss", metrics)

    def test_gan_generate(self):
        condition = _Tensor(np.zeros((12,), dtype=np.float32))
        plans = self.gan.generate(condition, num_samples=2)
        self.assertEqual(plans.shape, (2, PLAN_DIM))

    def test_gan_generate_2d_condition(self):
        condition = _Tensor(np.zeros((1, 12), dtype=np.float32))
        plans = self.gan.generate(condition, num_samples=1)
        self.assertEqual(plans.shape, (1, PLAN_DIM))

    def test_gan_save_and_load(self):
        tmpdir = tempfile.mkdtemp()
        path = os.path.join(tmpdir, "test_gan.pt")
        try:
            self.gan.save(path)
            self.assertTrue(os.path.exists(path))

            gan2 = ConditionalGAN(device="cpu")
            gan2.load(path)
            self.assertIsNotNone(gan2.generator)
            self.assertIsNotNone(gan2.discriminator)
        finally:
            import shutil

            shutil.rmtree(tmpdir)


class TestParseArgs(unittest.TestCase):
    def _parse(self, argv):
        with patch.object(sys, "argv", ["gan_train.py"] + argv):
            return parser.parse_args()

    def test_train_subcommand_defaults(self):
        args = self._parse(["train", "--dataset", DATA_CSV])
        self.assertEqual(args.command, "train")
        self.assertEqual(args.dataset, Path(DATA_CSV))
        self.assertEqual(args.epochs, 100)
        self.assertEqual(args.batch_size, 32)
        self.assertEqual(args.lr, 2e-4)
        self.assertFalse(args.cpu)
        self.assertEqual(args.output, Path("models") / GENERATOR_FILENAME)

    def test_train_subcommand_all_args(self):
        args = self._parse(
            [
                "train",
                "--dataset",
                DATA_CSV,
                "--epochs",
                "50",
                "--batch-size",
                "16",
                "--lr",
                "0.001",
                "--cpu",
                "--output",
                "models/my_model.pt",
            ]
        )
        self.assertEqual(args.command, "train")
        self.assertEqual(args.epochs, 50)
        self.assertEqual(args.batch_size, 16)
        self.assertEqual(args.lr, 0.001)
        self.assertTrue(args.cpu)
        self.assertEqual(args.output, Path("models/my_model.pt"))

    def test_generate_subcommand_defaults(self):
        args = self._parse(["generate", "--model", MODEL_PT, "--class-index", "2"])
        self.assertEqual(args.command, "generate")
        self.assertEqual(args.model, Path(MODEL_PT))
        self.assertEqual(args.class_index, 2)
        self.assertEqual(args.age, 30)
        self.assertEqual(args.gender, 1)
        self.assertEqual(args.weight, 75.0)
        self.assertEqual(args.height, 180.0)
        self.assertEqual(args.goal, 1)
        self.assertEqual(args.num_samples, 1)
        self.assertFalse(args.cpu)
        self.assertEqual(args.output, Path("models/generated_plan.json"))

    def test_generate_subcommand_all_args(self):
        args = self._parse(
            [
                "generate",
                "--model",
                "m.pt",
                "--class-index",
                "1",
                "--age",
                "40",
                "--gender",
                "0",
                "--weight",
                "80.5",
                "--height",
                "170",
                "--goal",
                "2",
                "--num-samples",
                "5",
                "--cpu",
                "--output",
                "out.json",
            ]
        )
        self.assertEqual(args.command, "generate")
        self.assertEqual(args.class_index, 1)
        self.assertEqual(args.age, 40.0)
        self.assertEqual(args.gender, 0.0)
        self.assertEqual(args.weight, 80.5)
        self.assertEqual(args.height, 170.0)
        self.assertEqual(args.goal, 2.0)
        self.assertEqual(args.num_samples, 5)
        self.assertTrue(args.cpu)
        self.assertEqual(args.output, Path("out.json"))


class TestTrainAndGenerate(unittest.TestCase):
    """Tests for train() and generate_plan() functions."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.models_dir = Path("models").resolve()
        self.models_dir.mkdir(parents=True, exist_ok=True)

        # Create a small dataset CSV
        plans_row = [float(i) for i in range(PLAN_DIM)]
        self.csv_path = os.path.join(self.tmpdir, "plans.csv")
        header = (
            "c0,c1,c2,c3,c4,c5,c6,c7,c8,c9,c10,c11,c12,c13,c14,c15,c16,c17,c18,"
            "klass,age,gender,weight,height,goal\n"
        )
        with open(self.csv_path, "w") as f:
            f.write(header)
            for i in range(4):
                row = plans_row + [i % 2, 30.0 + i, 1.0, 75.0, 180.0, 1.0]
                f.write(",".join(str(v) for v in row) + "\n")

    def tearDown(self):
        import shutil

        shutil.rmtree(self.tmpdir)
        if self.models_dir.exists():
            shutil.rmtree(self.models_dir)

    def _parse(self, argv):
        with patch.object(sys, "argv", ["gan_train.py"] + argv):
            return parser.parse_args()

    def test_train(self):
        args = self._parse(
            [
                "train",
                "--dataset",
                self.csv_path,
                "--epochs",
                "1",
                "--batch-size",
                "2",
                "--output",
                str(self.models_dir / GENERATOR_FILENAME),
            ]
        )
        train(args)
        self.assertTrue((self.models_dir / GENERATOR_FILENAME).exists())
        self.assertTrue((self.models_dir / "gan_norm_params.json").exists())
        self.assertTrue((self.models_dir / "gan_history.json").exists())

    def test_generate_plan(self):
        # First train a model so we have files to load
        args_train = self._parse(
            [
                "train",
                "--dataset",
                self.csv_path,
                "--epochs",
                "1",
                "--batch-size",
                "2",
                "--output",
                str(self.models_dir / GENERATOR_FILENAME),
            ]
        )
        train(args_train)

        # Now test generate_plan
        args_gen = self._parse(
            [
                "generate",
                "--model",
                str(self.models_dir / GENERATOR_FILENAME),
                "--class-index",
                "1",
                "--age",
                "30",
                "--gender",
                "1",
                "--weight",
                "75.0",
                "--height",
                "180.0",
                "--goal",
                "1",
                "--num-samples",
                "2",
                "--output",
                str(self.models_dir / "generated_plan.json"),
            ]
        )
        result = generate_plan(args_gen)
        self.assertEqual(len(result), 2)
        self.assertEqual(result[0]["class_index"], 1)
        self.assertEqual(len(result[0]["plan"]), PLAN_DIM)


if __name__ == "__main__":
    unittest.main()
