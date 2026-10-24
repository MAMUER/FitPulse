"""Tests for gan_train.py pure functions (no torch required at runtime).

Torch is not loadable on this system (DLL error), so we inject a fake torch
module into sys.modules before importing gan_train/gan_model to bypass the
top-level import. The tested functions only use numpy/stdlib.
"""

import json
import os
import sys
import tempfile
import types
import unittest
from pathlib import Path
from unittest.mock import patch

import numpy as np

# ---------------------------------------------------------------------------
# Inject a fake torch module so that importing gan_train / gan_model succeeds.
# The functions under test never touch torch at runtime.
# ---------------------------------------------------------------------------
_fake_torch = types.ModuleType("torch")
_fake_torch.device = lambda *a, **k: "cpu"
_fake_torch.FloatTensor = np.ndarray
_fake_torch.randn = lambda *a, **k: np.random.randn(*a).astype(np.float32)
_fake_torch.ones = lambda *a, **k: np.ones(a, dtype=np.float32)
_fake_torch.zeros = lambda *a, **k: np.zeros(a, dtype=np.float32)
_fake_torch.cuda = types.SimpleNamespace(is_available=lambda: False)
sys.modules["torch"] = _fake_torch

_fake_nn = types.ModuleType("torch.nn")
_fake_nn.Module = object
_fake_nn.Sequential = lambda *a, **k: None
_fake_nn.Linear = lambda *a, **k: None
_fake_nn.LeakyReLU = lambda *a, **k: None
_fake_nn.BatchNorm1d = lambda *a, **k: None
_fake_nn.Tanh = lambda *a, **k: None
_fake_nn.Dropout = lambda *a, **k: None
_fake_nn.Sigmoid = lambda *a, **k: None
_fake_nn.BCELoss = lambda *a, **k: None
sys.modules["torch.nn"] = _fake_nn

_fake_optim = types.ModuleType("torch.optim")
_fake_optim.Adam = lambda *a, **k: None
sys.modules["torch.optim"] = _fake_optim

_fake_utils = types.ModuleType("torch.utils")
_fake_data = types.ModuleType("torch.utils.data")
_fake_data.DataLoader = lambda *a, **k: None
_fake_data.Dataset = object
sys.modules["torch.utils"] = _fake_utils
sys.modules["torch.utils.data"] = _fake_data

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from gan_train import _safe_path, load_dataset, parse_args, parser  # noqa: E402
from gan_model import (  # noqa: E402
    build_condition,
    denormalize_plan,
    normalize_plan,
    CONDITION_DIM,
    NUM_CLASSES,
    PLAN_DIM,
)


class TestSafePath(unittest.TestCase):
    """Tests for _safe_path path traversal protection."""

    def setUp(self):
        self.models_dir = Path("models").resolve()
        self.models_dir.mkdir(parents=True, exist_ok=True)

    def tearDown(self):
        import shutil
        if self.models_dir.exists():
            shutil.rmtree(self.models_dir)

    def test_valid_path_inside_models(self):
        """A path inside models/ should be returned resolved."""
        p = self.models_dir / "gan_generator.pt"
        result = _safe_path(p)
        self.assertEqual(result, p.resolve())

    def test_path_traversal_blocked(self):
        """A path outside models/ should raise ValueError."""
        p = Path("outside") / "model.pt"
        with self.assertRaises(ValueError):
            _safe_path(p)

    def test_traversal_with_dotdot(self):
        """Path with .. escaping models/ should raise ValueError."""
        p = self.models_dir / ".." / "escape.pt"
        with self.assertRaises(ValueError):
            _safe_path(p)


class TestLoadDataset(unittest.TestCase):
    """Tests for load_dataset CSV loading."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def _write_csv(self, rows):
        """Write CSV with ASCII header (np.loadtxt needs numeric data after header)."""
        path = os.path.join(self.tmpdir, "plans.csv")
        with open(path, "w") as f:
            f.write("c0,c1,c2,c3,c4,c5,c6,c7,c8,c9,c10,c11,c12,c13,c14,c15,c16,c17,c18,klass,age,gender,weight,height,goal\n")
            for row in rows:
                f.write(",".join(str(v) for v in row) + "\n")
        return path

    def test_valid_dataset(self):
        """A valid CSV should load plans and conditions."""
        plans_row = [float(i) for i in range(PLAN_DIM)]
        row1 = plans_row + [0, 30.0, 1.0, 75.0, 180.0, 1.0]
        row2 = plans_row + [1, 25.0, 0.0, 60.0, 165.0, 2.0]
        path = self._write_csv([row1, row2])

        plans, conditions = load_dataset(path)

        self.assertEqual(plans.shape, (2, PLAN_DIM))
        self.assertEqual(conditions.shape[0], 2)
        self.assertEqual(conditions.shape[1], CONDITION_DIM)

    def test_missing_file(self):
        """A missing file should raise FileNotFoundError."""
        path = os.path.join(self.tmpdir, "nonexistent.csv")
        with self.assertRaises(FileNotFoundError):
            load_dataset(path)

    def test_negative_plan_values(self):
        """Negative plan values should raise ValueError."""
        plans_row = [float(i) for i in range(PLAN_DIM)]
        plans_row[0] = -1.0
        row1 = plans_row + [0, 30.0, 1.0, 75.0, 180.0, 1.0]
        row2 = plans_row + [1, 25.0, 0.0, 60.0, 165.0, 2.0]
        path = self._write_csv([row1, row2])

        with self.assertRaises(ValueError):
            load_dataset(path)

    def test_negative_user_features(self):
        """Negative user features should raise ValueError."""
        plans_row = [float(i) for i in range(PLAN_DIM)]
        row1 = plans_row + [0, -30.0, 1.0, 75.0, 180.0, 1.0]
        row2 = plans_row + [1, 25.0, 0.0, 60.0, 165.0, 2.0]
        path = self._write_csv([row1, row2])

        with self.assertRaises(ValueError):
            load_dataset(path)


class TestBuildCondition(unittest.TestCase):
    """Tests for build_condition one-hot encoding."""

    def test_one_hot_encoding(self):
        """Condition should have one-hot class + user features."""
        features = np.array([30.0, 1.0, 75.0, 180.0, 1.0], dtype=np.float32)
        cond = build_condition(2, features)

        self.assertEqual(cond.shape, (CONDITION_DIM,))
        self.assertEqual(cond[2], 1.0)
        self.assertEqual(np.sum(cond[:NUM_CLASSES]), 1.0)
        np.testing.assert_array_equal(cond[NUM_CLASSES:], features)

    def test_class_index_zero(self):
        """Class index 0 should set first one-hot element."""
        features = np.array([25.0, 0.0, 60.0, 165.0, 2.0], dtype=np.float32)
        cond = build_condition(0, features)
        self.assertEqual(cond[0], 1.0)
        self.assertEqual(np.sum(cond[:NUM_CLASSES]), 1.0)

    def test_class_index_max(self):
        """Class index 6 (last) should set last one-hot element."""
        features = np.array([40.0, 1.0, 80.0, 190.0, 3.0], dtype=np.float32)
        cond = build_condition(6, features)
        self.assertEqual(cond[6], 1.0)
        self.assertEqual(np.sum(cond[:NUM_CLASSES]), 1.0)


class TestNormalizePlan(unittest.TestCase):
    """Tests for normalize_plan normalization to [-1, 1]."""

    def test_normalize_basic(self):
        """Values should be normalized to [-1, 1] range."""
        plan = np.array([0.0, 50.0, 100.0], dtype=np.float32)
        plan_min = np.array([0.0, 0.0, 0.0], dtype=np.float32)
        plan_max = np.array([100.0, 100.0, 100.0], dtype=np.float32)

        result = normalize_plan(plan, plan_min, plan_max)

        np.testing.assert_almost_equal(result[0], -1.0)
        np.testing.assert_almost_equal(result[1], 0.0)
        np.testing.assert_almost_equal(result[2], 1.0)

    def test_normalize_identical_min_max(self):
        """When min == max, result should be -1 (due to eps)."""
        plan = np.array([5.0, 5.0], dtype=np.float32)
        plan_min = np.array([5.0, 5.0], dtype=np.float32)
        plan_max = np.array([5.0, 5.0], dtype=np.float32)

        result = normalize_plan(plan, plan_min, plan_max)

        np.testing.assert_almost_equal(result[0], -1.0)
        np.testing.assert_almost_equal(result[1], -1.0)

    def test_normalize_out_of_range(self):
        """Values outside [min, max] can exceed [-1, 1]."""
        plan = np.array([200.0], dtype=np.float32)
        plan_min = np.array([0.0], dtype=np.float32)
        plan_max = np.array([100.0], dtype=np.float32)

        result = normalize_plan(plan, plan_min, plan_max)

        self.assertGreater(result[0], 1.0)


class TestDenormalizePlan(unittest.TestCase):
    """Tests for denormalize_plan back to original range.

    Note: denormalize_plan clips to [0.0, 1.0] (not [plan_min, plan_max]),
    so values are always in [0, 1] after denormalization.
    """

    def test_denormalize_basic(self):
        """Values should be denormalized from [-1, 1] to [0, 1] range."""
        plan_norm = np.array([-1.0, 0.0, 1.0], dtype=np.float32)
        plan_min = np.array([0.0, 0.0, 0.0], dtype=np.float32)
        plan_max = np.array([100.0, 100.0, 100.0], dtype=np.float32)

        result = denormalize_plan(plan_norm, plan_min, plan_max)

        # (plan_norm + 1) / 2 * (plan_max - plan_min) + plan_min
        # -1 -> 0, 0 -> 50, 1 -> 100, then clip to [0, 1]
        np.testing.assert_almost_equal(result[0], 0.0)
        np.testing.assert_almost_equal(result[1], 1.0)  # 50 clipped to 1
        np.testing.assert_almost_equal(result[2], 1.0)  # 100 clipped to 1

    def test_denormalize_clips_above_one(self):
        """Values above 1.0 are clipped to 1.0."""
        plan_norm = np.array([2.0], dtype=np.float32)
        plan_min = np.array([0.0], dtype=np.float32)
        plan_max = np.array([100.0], dtype=np.float32)

        result = denormalize_plan(plan_norm, plan_min, plan_max)
        self.assertEqual(float(result[0]), 1.0)

    def test_denormalize_clips_below_zero(self):
        """Values below 0.0 are clipped to 0.0."""
        plan_norm = np.array([-2.0], dtype=np.float32)
        plan_min = np.array([0.0], dtype=np.float32)
        plan_max = np.array([100.0], dtype=np.float32)

        result = denormalize_plan(plan_norm, plan_min, plan_max)
        self.assertEqual(float(result[0]), 0.0)


class TestParseArgs(unittest.TestCase):
    """Tests for parse_args CLI argument parsing.

    parse_args() reads from sys.argv, so we patch sys.argv.
    """

    def _parse(self, argv):
        with patch.object(sys, "argv", ["gan_train.py"] + argv):
            return parser.parse_args()

    def test_train_subcommand_defaults(self):
        """Train subcommand should have correct defaults."""
        args = self._parse(["train", "--dataset", "data.csv"])
        self.assertEqual(args.command, "train")
        self.assertEqual(args.dataset, Path("data.csv"))
        self.assertEqual(args.epochs, 100)
        self.assertEqual(args.batch_size, 32)
        self.assertEqual(args.lr, 2e-4)
        self.assertFalse(args.cpu)
        self.assertEqual(args.output, Path("models/gan_generator.pt"))

    def test_train_subcommand_all_args(self):
        """Train subcommand should parse all arguments."""
        args = self._parse([
            "train", "--dataset", "data.csv",
            "--epochs", "50",
            "--batch-size", "16",
            "--lr", "0.001",
            "--cpu",
            "--output", "models/my_model.pt",
        ])
        self.assertEqual(args.command, "train")
        self.assertEqual(args.epochs, 50)
        self.assertEqual(args.batch_size, 16)
        self.assertEqual(args.lr, 0.001)
        self.assertTrue(args.cpu)
        self.assertEqual(args.output, Path("models/my_model.pt"))

    def test_generate_subcommand_defaults(self):
        """Generate subcommand should have correct defaults."""
        args = self._parse(["generate", "--model", "model.pt", "--class-index", "2"])
        self.assertEqual(args.command, "generate")
        self.assertEqual(args.model, Path("model.pt"))
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
        """Generate subcommand should parse all arguments."""
        args = self._parse([
            "generate", "--model", "m.pt", "--class-index", "1",
            "--age", "40", "--gender", "0", "--weight", "80.5",
            "--height", "170", "--goal", "2",
            "--num-samples", "5", "--cpu",
            "--output", "out.json",
        ])
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


if __name__ == "__main__":
    unittest.main()