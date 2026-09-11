"""
Unit tests for cmd/ml_generator/train_gan.py

Tests cover:
- ConditionalDiffusionModel init, forward, and sample
- build_rule_based_plan / build_static_beginner_plan
- apply_post_processing_rules
- encode_user_profile / decode_plan

Requires pytest and torch (CPU-only).
Run from repo root or cmd/ml_generator/:
    pytest cmd/ml_generator/tests/test_train_gan.py -v
"""

import sys
import types
from unittest import mock

import numpy as np
import pytest
import torch  # type: ignore

# ---------------------------------------------------------------------------
# Lightweight module-level mocks so we can import train_gan without wandb /
# lightning being fully installed.  We insert fake modules into sys.modules
# *before* importing train_gan.
# ---------------------------------------------------------------------------


class _MockWandb:
    """Stand-in for wandb when WANDB_ENABLED is True."""

    @staticmethod
    def init(**kwargs):
        return None

    @staticmethod
    def log(*args, **kwargs):
        return None

    class config:
        @staticmethod
        def update(*args, **kwargs):
            return None

    @staticmethod
    def finish():
        return None


class _MockLightningModule:
    """Stand-in for lightning.LightningModule."""

    def __init__(self, *args, **kwargs):
        pass

    def __call__(self, *args, **kwargs):
        return None

    def __setattr__(self, name, value):
        object.__setattr__(self, name, value)

    def __getattr__(self, name):
        return mock.MagicMock()

    def configure_optimizers(self):
        return None

    def training_step(self, batch, batch_idx):
        return None

    def validation_step(self, batch, batch_idx):
        return None

    def log(self, *args, **kwargs):
        return None


class _MockTrainer:
    """Stand-in for lightning.Trainer."""

    def __init__(self, *args, **kwargs):
        self.callback_metrics = {}
        self.current_epoch = 0

    def fit(self, model, train_loader=None, val_loader=None):
        return None

    def validate(self, model=None, dataloaders=None):
        return None


class _MockEarlyStopping:
    def __init__(self, *args, **kwargs):
        pass


class _MockModelCheckpoint:
    def __init__(self, *args, **kwargs):
        self.best_model_path = ""
        self.best_model_score = None


class _MockL:
    """Stand-in for the top-level `lightning` package."""

    pytorch = types.SimpleNamespace(
        callbacks=types.SimpleNamespace(
            EarlyStopping=_MockEarlyStopping,
            ModelCheckpoint=_MockModelCheckpoint,
        ),
    )
    Trainer = _MockTrainer


def _install_mocks():
    """Insert mock modules into sys.modules before importing train_gan."""
    sys.modules.setdefault("wandb", _MockWandb())
    sys.modules.setdefault("lightning", _MockL())
    sys.modules.setdefault("lightning.pytorch", sys.modules["lightning"].pytorch)
    sys.modules.setdefault(
        "lightning.pytorch.callbacks",
        sys.modules["lightning"].pytorch.callbacks,
    )


_install_mocks()

# Now import the module under test
from cmd.ml_generator import train_gan  # noqa: E402  (import after mocks)

# ===========================================================================
# Fixtures
# ===========================================================================


@pytest.fixture(autouse=True)
def _isolate_global_state():
    """Reset the module-level betas_np / _betas_np before each test."""
    train_gan._betas_np = None
    train_gan.betas_np = train_gan.get_betas()
    yield


@pytest.fixture()
def default_model():
    """Return a fresh ConditionalDiffusionModel on CPU."""
    torch.manual_seed(0)
    return train_gan.ConditionalDiffusionModel()


# ===========================================================================
# ConditionalDiffusionModel
# ===========================================================================


class TestConditionalDiffusionModelInit:
    def test_default_dimensions(self, default_model):
        assert default_model.latent_dim == train_gan.LATENT_DIM
        assert default_model.plan_dim == train_gan.PLAN_DIM
        assert default_model.condition_dim == train_gan.CONDITION_DIM

    def test_custom_dimensions(self):
        model = train_gan.ConditionalDiffusionModel(
            latent_dim=32,
            plan_dim=10,
            condition_dim=8,
        )
        assert model.latent_dim == 32
        assert model.plan_dim == 10
        assert model.condition_dim == 8

    def test_noise_pred_net_input_dimension(self, default_model):
        """Input should be plan_dim + condition_dim + 1 (time embedding)."""
        net = default_model.noise_pred_net
        # First Linear layer
        first_linear = net[0]
        assert isinstance(first_linear, torch.nn.Linear)
        expected_in = train_gan.PLAN_DIM + train_gan.CONDITION_DIM + 1
        assert first_linear.in_features == expected_in

    def test_noise_pred_net_output_dimension(self, default_model):
        """Output should be plan_dim."""
        net = default_model.noise_pred_net
        last_linear = net[-1]
        assert isinstance(last_linear, torch.nn.Linear)
        assert last_linear.out_features == train_gan.PLAN_DIM

    def test_betas_registered(self, default_model):
        assert len(default_model.betas) == 1000
        assert default_model.betas[0].item() == pytest.approx(1e-4)
        assert default_model.betas[-1].item() == pytest.approx(0.02)

    def test_alphas_are_complement(self, default_model):
        alphas = default_model.alphas.detach()
        betas = default_model.betas.detach()
        assert torch.allclose(alphas, 1.0 - betas, atol=1e-7)

    def test_alpha_bar_is_cumprod(self, default_model):
        expected = torch.cumprod(default_model.alphas.detach(), dim=0)
        assert torch.allclose(default_model.alpha_bar.detach(), expected, atol=1e-7)


class TestConditionalDiffusionModelForward:
    def test_output_shape_default_condition(self, default_model):
        default_model.eval()
        batch = 4
        x_t = torch.randn(batch, train_gan.PLAN_DIM)
        t = torch.randint(0, 1000, (batch,)).float() / 1000.0
        with torch.no_grad():
            out = default_model(x_t, t)
        assert out.shape == (batch, train_gan.PLAN_DIM)

    def test_output_shape_with_condition(self, default_model):
        default_model.eval()
        batch = 2
        x_t = torch.randn(batch, train_gan.PLAN_DIM)
        t = torch.rand(batch)
        cond = torch.randn(batch, train_gan.CONDITION_DIM)
        with torch.no_grad():
            out = default_model(x_t, t, cond)
        assert out.shape == (batch, train_gan.PLAN_DIM)

    def test_forward_is_deterministic_no_condition(self, default_model):
        default_model.eval()
        x_t = torch.randn(3, train_gan.PLAN_DIM)
        t = torch.full((3,), 0.5)
        with torch.no_grad():
            out1 = default_model(x_t, t)
            out2 = default_model(x_t, t)
        assert torch.allclose(out1, out2)

    def test_forward_uses_condition_tensor(self, default_model):
        default_model.eval()
        x_t = torch.zeros(1, train_gan.PLAN_DIM)
        t = torch.zeros(1)
        cond_a = torch.zeros(1, train_gan.CONDITION_DIM)
        cond_b = torch.ones(1, train_gan.CONDITION_DIM)
        with torch.no_grad():
            out_a = default_model(x_t, t, cond_a)
            out_b = default_model(x_t, t, cond_b)
        assert not torch.allclose(out_a, out_b)


class TestConditionalDiffusionModelSample:
    def test_output_shape(self, default_model):
        default_model.eval()
        with torch.no_grad():
            sample = default_model.sample(num_steps=10)
        assert sample.shape == (1, train_gan.PLAN_DIM)

    def test_output_range_in_unit_interval(self, default_model):
        default_model.eval()
        torch.manual_seed(42)
        with torch.no_grad():
            sample = default_model.sample(num_steps=10)
        assert (sample >= 0.0).all(), "sample values should be >= 0"
        assert (sample <= 1.0).all(), "sample values should be <= 1"

    def test_output_dtype_float(self, default_model):
        default_model.eval()
        with torch.no_grad():
            sample = default_model.sample(num_steps=10)
        assert sample.dtype == torch.float32

    def test_sample_different_seeds_produce_different_outputs(self, default_model):
        default_model.eval()
        with torch.no_grad():
            torch.manual_seed(0)
            s1 = default_model.sample(num_steps=5).clone()
            torch.manual_seed(99)
            s2 = default_model.sample(num_steps=5).clone()
        assert not torch.allclose(s1, s2)

    def test_sample_with_explicit_condition(self, default_model):
        default_model.eval()
        cond = torch.randn(1, train_gan.CONDITION_DIM)
        with torch.no_grad():
            sample = default_model.sample(condition=cond, num_steps=10)
        assert sample.shape == (1, train_gan.PLAN_DIM)

    def test_sample_t0_uses_correct_alpha_bar(self, default_model):
        """At t=0 the reverse step should use alpha_bar_prev = 1.0 (no noise)."""
        default_model.eval()
        with torch.no_grad():
            sample = default_model.sample(num_steps=1)
        assert (sample >= 0.0).all() and (sample <= 1.0).all()


# ===========================================================================
# Rule-based plan builders
# ===========================================================================


class TestBuildRuleBasedPlan:
    def test_output_shape(self):
        profile = train_gan.UserProfile(age=30, fitness_level="intermediate")
        plan = train_gan.build_rule_based_plan("endurance_basic", profile)
        assert plan.shape == (train_gan.PLAN_DIM,)

    def test_output_dtype(self):
        profile = train_gan.UserProfile()
        plan = train_gan.build_rule_based_plan("recovery", profile)
        assert plan.dtype == np.float32

    def test_all_values_in_unit_interval(self):
        profile = train_gan.UserProfile(age=25, fitness_level="advanced")
        for cls in train_gan.TRAINING_TEMPLATES:
            plan = train_gan.build_rule_based_plan(cls, profile)
            assert (plan >= 0.0).all() and (
                plan <= 1.0
            ).all(), f"values out of range for class {cls}"

    def test_unknown_class_falls_back_to_endurance_basic(self):
        profile = train_gan.UserProfile()
        plan_known = train_gan.build_rule_based_plan("endurance_basic", profile)
        plan_unknown = train_gan.build_rule_based_plan("nonexistent_class", profile)
        np.testing.assert_array_equal(plan_known, plan_unknown)

    def test_muscle_gain_goal_increases_strength_dim(self):
        profile = train_gan.UserProfile(goals=["набор массы"])
        plan = train_gan.build_rule_based_plan("endurance_basic", profile)
        # index 14 is goal_strength
        assert plan[14] == pytest.approx(0.8)

    def test_endurance_goal_increases_endurance_dim(self):
        profile = train_gan.UserProfile(goals=["выносливость"])
        plan = train_gan.build_rule_based_plan("endurance_basic", profile)
        # index 15 is goal_endurance
        assert plan[15] == pytest.approx(0.8)

    def test_no_goals_sets_low_strength_and_endurance(self):
        profile = train_gan.UserProfile(goals=[])
        plan = train_gan.build_rule_based_plan("endurance_basic", profile)
        assert plan[14] == pytest.approx(0.2)
        assert plan[15] == pytest.approx(0.2)

    def test_age_factor_declines_with_age(self):
        young = train_gan.UserProfile(age=25)
        old = train_gan.UserProfile(age=70)
        plan_young = train_gan.build_rule_based_plan("endurance_basic", young)
        plan_old = train_gan.build_rule_based_plan("endurance_basic", old)
        # index 16 is age_factor
        assert plan_young[16] >= plan_old[16]

    def test_beginner_fitness_lower_than_advanced(self):
        beginner = train_gan.UserProfile(fitness_level="beginner")
        advanced = train_gan.UserProfile(fitness_level="advanced")
        plan_beginner = train_gan.build_rule_based_plan("power_hiit", beginner)
        plan_advanced = train_gan.build_rule_based_plan("power_hiit", advanced)
        # index 17 is fitness_factor
        assert plan_beginner[17] < plan_advanced[17]

    def test_health_conditions_reduce_health_factor(self):
        no_conditions = train_gan.UserProfile(health_conditions=[])
        with_conditions = train_gan.UserProfile(health_conditions=["asthma"])
        plan_no = train_gan.build_rule_based_plan("endurance_basic", no_conditions)
        plan_yes = train_gan.build_rule_based_plan("endurance_basic", with_conditions)
        # index 18 is health_factor
        assert plan_no[18] > plan_yes[18]

    def test_first_three_dims_non_zero_for_valid_class(self):
        profile = train_gan.UserProfile()
        plan = train_gan.build_rule_based_plan("power_hiit", profile)
        assert plan[0] > 0.0  # duration
        assert plan[1] > 0.0  # intensity
        assert plan[2] > 0.0  # rest_ratio


class TestBuildStaticBeginnerPlan:
    def test_output_shape(self):
        plan = train_gan.build_static_beginner_plan()
        assert plan.shape == (train_gan.PLAN_DIM,)

    def test_output_dtype(self):
        plan = train_gan.build_static_beginner_plan()
        assert plan.dtype == np.float32

    def test_values_in_unit_interval(self):
        plan = train_gan.build_static_beginner_plan()
        assert (plan >= 0.0).all() and (plan <= 1.0).all()

    def test_is_deterministic(self):
        p1 = train_gan.build_static_beginner_plan()
        p2 = train_gan.build_static_beginner_plan()
        np.testing.assert_array_equal(p1, p2)

    def test_known_values(self):
        plan = train_gan.build_static_beginner_plan()
        assert plan[0] == pytest.approx(0.4)
        assert plan[3] == pytest.approx(0.4)
        assert plan[11] == pytest.approx(0.0)
        assert plan[18] == pytest.approx(0.2)


# ===========================================================================
# apply_post_processing_rules
# ===========================================================================


class TestApplyPostProcessingRules:
    def _base_plan(self):
        return np.ones(train_gan.PLAN_DIM, dtype=np.float32)

    def test_returns_copy_not_reference(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        assert result is not plan

    def test_menstruation_reduces_intensity(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(),
            health_status=train_gan.HealthStatus(menstrual_phase="menstruation"),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        # index 1 is intensity; should be reduced by 0.2
        assert result[1] == pytest.approx(0.8)

    def test_pregnancy_contraindication_caps_intensity_and_duration(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(contraindications=["pregnancy"]),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        assert result[1] <= 0.5  # intensity capped at 0.5
        assert result[0] <= 0.4  # duration capped at 0.4

    def test_illness_class_zeroes_duration_and_intensity(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="illness",
            user_profile=train_gan.UserProfile(),
            health_status=train_gan.HealthStatus(predicted_class="illness"),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        assert result[0] == 0.0
        assert result[1] == 0.0

    def test_overtraining_class_reduces_weekly_freq_and_increases_rest(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="overtraining",
            user_profile=train_gan.UserProfile(),
            health_status=train_gan.HealthStatus(predicted_class="overtraining"),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        # index 3 = weekly_freq, index 2 = rest_ratio
        assert result[3] == pytest.approx(max(0.2, 1.0 * 0.7))
        assert result[2] == pytest.approx(min(1.0, 1.0 + 0.2))

    def test_senior_age_reduces_intensity(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(age=65),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        assert result[1] == pytest.approx(0.8)

    def test_high_bmi_reduces_intensity(self):
        plan = self._base_plan()
        # bmi = 70 / (1.5)^2 ≈ 31.1 -> not triggered
        profile_normal = train_gan.UserProfile(weight=70, height=150)
        # bmi = 200 / (1.4)^2 ≈ 102 -> triggered
        profile_obese = train_gan.UserProfile(weight=200, height=140)
        request_normal = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=profile_normal,
        )
        request_obese = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=profile_obese,
        )
        result_normal = train_gan.apply_post_processing_rules(
            plan.copy(), request_normal
        )
        result_obese = train_gan.apply_post_processing_rules(plan.copy(), request_obese)
        assert result_obese[1] <= result_normal[1]

    def test_low_sleep_reduces_intensity(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(),
            health_status=train_gan.HealthStatus(sleep_hours=5.0),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        assert result[1] == pytest.approx(0.85)

    def test_no_conditions_plan_unchanged(self):
        plan = self._base_plan()
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        np.testing.assert_array_equal(result, plan)

    def test_intensity_never_goes_below_zero(self):
        plan = self._base_plan()
        # Set intensity to 0.01; applying menustration should clamp to >= 0
        plan[1] = 0.01
        request = train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(),
            health_status=train_gan.HealthStatus(menstrual_phase="menstruation"),
        )
        result = train_gan.apply_post_processing_rules(plan, request)
        assert result[1] >= 0.0


# ===========================================================================
# encode_user_profile
# ===========================================================================


class TestEncodeUserProfile:
    def test_output_shape(self):
        profile = train_gan.UserProfile()
        encoded = train_gan.encode_user_profile(profile)
        assert encoded.shape == (1, train_gan.CONDITION_DIM)

    def test_output_dtype(self):
        profile = train_gan.UserProfile()
        encoded = train_gan.encode_user_profile(profile)
        assert encoded.dtype == np.float32

    def test_output_values_in_unit_interval(self):
        profile = train_gan.UserProfile()
        encoded = train_gan.encode_user_profile(profile)
        assert (encoded >= 0.0).all() and (encoded <= 1.0).all()

    def test_muscle_gain_goal_sets_strength_bit(self):
        profile = train_gan.UserProfile(goals=["набор массы"])
        encoded = train_gan.encode_user_profile(profile)
        # goals start at index 3: [strength, endurance, weight_loss, flexibility]
        assert encoded[0, 3] == 1.0  # goal_strength

    def test_no_goals_sets_all_goal_bits_zero(self):
        profile = train_gan.UserProfile(goals=[])
        encoded = train_gan.encode_user_profile(profile)
        assert encoded[0, 3] == 0.0
        assert encoded[0, 4] == 0.0
        assert encoded[0, 5] == 0.0
        assert encoded[0, 6] == 0.0

    def test_illness_predicted_class_sets_recovery_bit(self):
        health = train_gan.HealthStatus(predicted_class="recovery")
        encoded = train_gan.encode_user_profile(
            train_gan.UserProfile(), health_status=health
        )
        # recovery_needed is index 14
        assert encoded[0, 14] == 1.0

    def test_contraindications_set_bit(self):
        profile = train_gan.UserProfile(contraindications=["diabetes"])
        encoded = train_gan.encode_user_profile(profile)
        # has_contraindications is index 12
        assert encoded[0, 12] == 1.0

    def test_allergies_set_bit(self):
        profile = train_gan.UserProfile(allergies=["nuts"])
        encoded = train_gan.encode_user_profile(profile)
        # has_allergies is index 13
        assert encoded[0, 13] == 1.0

    def test_age_encoding_increases_with_age(self):
        young = train_gan.UserProfile(age=20)
        old = train_gan.UserProfile(age=80)
        enc_young = train_gan.encode_user_profile(young)
        enc_old = train_gan.encode_user_profile(old)
        # age_norm is index 0; older -> higher value (closer to 1)
        assert enc_old[0, 0] > enc_young[0, 0]

    def test_none_preferences_defaults(self):
        profile = train_gan.UserProfile()
        encoded = train_gan.encode_user_profile(profile, preferences=None)
        assert encoded.shape == (1, train_gan.CONDITION_DIM)

    def test_with_all_fields_provided(self):
        profile = train_gan.UserProfile(
            age=28,
            gender="female",
            fitness_level="advanced",
            weight=60.0,
            height=165.0,
            health_conditions=["asthma"],
            goals=["выносливость", "похудение"],
            allergies=["латекс"],
            contraindications=["hypertension"],
        )
        health = train_gan.HealthStatus(
            predicted_class="endurance_basic",
            confidence=0.9,
            hrv=80.0,
            sleep_hours=8.0,
            active_conditions_count=1,
            menstrual_phase="follicular",
            day_of_cycle=10,
            cycle_length=28,
            body_composition={"temperature": 36.6, "spo2": 98.0},
        )
        history = train_gan.TrainingHistory(
            completed_workouts_count=10,
            avg_intensity=0.7,
            last_workout_date="2025-01-15T00:00:00Z",
        )
        preferences = {
            "available_days": ["mon", "wed", "fri"],
            "time": "morning",
            "equipment": ["dumbbell"],
        }
        encoded = train_gan.encode_user_profile(profile, health, history, preferences)
        assert encoded.shape == (1, train_gan.CONDITION_DIM)


# ===========================================================================
# decode_plan
# ===========================================================================


class TestDecodePlan:
    def _default_request(self):
        return train_gan.PlanGenerationRequest(
            training_class="endurance_basic",
            user_profile=train_gan.UserProfile(),
        )

    def test_output_has_required_keys(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(
            plan_vec, "endurance_basic", train_gan.UserProfile()
        )
        expected_keys = {
            "training_type",
            "training_type_ru",
            "duration_minutes",
            "intensity",
            "weekly_frequency",
            "primary_exercise",
            "warmup_minutes",
            "cooldown_minutes",
            "exercises",
            "session_structure",
            "notes",
            "weekly_schedule",
        }
        assert expected_keys.issubset(result.keys())

    def test_duration_minutes_in_valid_range(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(
            plan_vec, "endurance_basic", train_gan.UserProfile()
        )
        assert 20 <= result["duration_minutes"] <= 120

    def test_intensity_in_unit_interval(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(
            plan_vec, "endurance_basic", train_gan.UserProfile()
        )
        assert 0.0 <= result["intensity"] <= 1.0

    def test_weekly_frequency_in_valid_range(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(
            plan_vec, "endurance_basic", train_gan.UserProfile()
        )
        assert 1 <= result["weekly_frequency"] <= 7

    def test_unknown_class_falls_back(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(
            plan_vec, "nonexistent_class", train_gan.UserProfile()
        )
        assert result["training_type"] == "nonexistent_class"
        assert (
            result["training_type_ru"]
            == train_gan.TRAINING_TEMPLATES["endurance_basic"]["name_ru"]
        )

    def test_session_structure_has_three_exercises(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "power_hiit", train_gan.UserProfile())
        assert len(result["session_structure"]) == 3

    def test_weekly_schedule_has_expected_days(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(
            plan_vec, "endurance_basic", train_gan.UserProfile()
        )
        expected_days = {"monday", "wednesday", "friday", "saturday", "sunday"}
        assert expected_days == set(result["weekly_schedule"].keys())

    def test_beginner_gets_duration_reduction_note(self):
        profile = train_gan.UserProfile(fitness_level="beginner")
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "endurance_basic", profile)
        notes_text = " ".join(result["notes"])
        assert "интенсивности" in notes_text or "50%" in notes_text

    def test_senior_gets_warmup_note(self):
        profile = train_gan.UserProfile(age=55)
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "endurance_basic", profile)
        notes_text = " ".join(result["notes"])
        assert "разминки" in notes_text

    def test_weight_loss_goal_adds_cardio_note(self):
        profile = train_gan.UserProfile(goals=["похудение"])
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "endurance_basic", profile)
        notes_text = " ".join(result["notes"])
        assert "кардио" in notes_text

    def test_muscle_gain_goal_adds_strength_note(self):
        profile = train_gan.UserProfile(goals=["набор массы"])
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "endurance_basic", profile)
        notes_text = " ".join(result["notes"])
        assert "силовых" in notes_text

    def test_rehab_goal_adds_technique_note(self):
        profile = train_gan.UserProfile(goals=["реабилитация"])
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "endurance_basic", profile)
        notes_text = " ".join(result["notes"])
        assert "техникой" in notes_text

    def test_health_conditions_adds_doctor_note(self):
        profile = train_gan.UserProfile(health_conditions=["asthma"])
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "endurance_basic", profile)
        notes_text = " ".join(result["notes"])
        assert "врач" in notes_text

    def test_primary_exercise_from_template(self):
        plan_vec = np.ones(train_gan.PLAN_DIM, dtype=np.float32)
        result = train_gan.decode_plan(plan_vec, "power_hiit", train_gan.UserProfile())
        assert (
            result["primary_exercise"]
            in train_gan.TRAINING_TEMPLATES["power_hiit"]["exercises"]
        )


# ===========================================================================
# load_real_data
# ===========================================================================


class TestLoadRealData:
    def test_raises_file_not_found_when_no_data_files(self, tmp_path, monkeypatch):
        monkeypatch.setattr(
            train_gan, "TRAINING_DATA_PATH", tmp_path / "nonexistent.csv"
        )
        monkeypatch.setattr(
            train_gan, "FALLBACK_TRAINING_DATA_PATH", tmp_path / "fallback.csv"
        )
        with pytest.raises(FileNotFoundError):
            train_gan.load_real_data()

    def test_uses_fallback_when_primary_missing(self, tmp_path, monkeypatch):
        import pandas as pd

        fallback_path = tmp_path / "fallback.csv"
        plan_vec_str = (
            "[0.5, 0.5, 0.5, 0.5, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, "
            "0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]"
        )
        df = pd.DataFrame({"plan_vector": [plan_vec_str] * 10})
        df.to_csv(fallback_path, index=False)

        monkeypatch.setattr(train_gan, "TRAINING_DATA_PATH", tmp_path / "primary.csv")
        monkeypatch.setattr(train_gan, "FALLBACK_TRAINING_DATA_PATH", fallback_path)

        train, train_c, val, val_c = train_gan.load_real_data()
        assert len(train) + len(val) == 10
        assert train_c.shape[1] == train_gan.CONDITION_DIM
        assert val_c.shape[1] == train_gan.CONDITION_DIM
        assert (train_c == 0).all()  # zero conditions in fallback mode

    def test_normalizes_plan_values_to_minus_one_to_one(self, tmp_path, monkeypatch):
        import pandas as pd

        data_path = tmp_path / "training_data.csv"
        # Values in [0, 1] -> should be normalized to [-1, 1]
        vec = [0.0, 0.5, 1.0] + [0.0] * (train_gan.PLAN_DIM - 3)
        df = pd.DataFrame({"plan_vector": [str(vec)] * 8})
        df.to_csv(data_path, index=False)

        monkeypatch.setattr(train_gan, "TRAINING_DATA_PATH", data_path)
        monkeypatch.setattr(
            train_gan, "FALLBACK_TRAINING_DATA_PATH", tmp_path / "fallback.csv"
        )

        train, _, _, _ = train_gan.load_real_data()
        # 0.0 -> -1.0; 0.5 -> 0.0; 1.0 -> 1.0
        assert train[0, 0] == pytest.approx(-1.0)
        assert train[0, 1] == pytest.approx(0.0)
        assert train[0, 2] == pytest.approx(1.0)

    def test_80_20_train_val_split(self, tmp_path, monkeypatch):
        import pandas as pd

        data_path = tmp_path / "training_data.csv"
        vec = [0.5] * train_gan.PLAN_DIM
        df = pd.DataFrame({"plan_vector": [str(vec)] * 100})
        df.to_csv(data_path, index=False)

        monkeypatch.setattr(train_gan, "TRAINING_DATA_PATH", data_path)
        monkeypatch.setattr(
            train_gan, "FALLBACK_TRAINING_DATA_PATH", tmp_path / "fallback.csv"
        )

        train, _, val, _ = train_gan.load_real_data()
        assert len(train) == 80
        assert len(val) == 20

    def test_loads_condition_vector_column(self, tmp_path, monkeypatch):
        import pandas as pd

        data_path = tmp_path / "training_data.csv"
        plan_vec = [0.5] * train_gan.PLAN_DIM
        cond_vec = [float(i % 2) for i in range(train_gan.CONDITION_DIM)]
        df = pd.DataFrame(
            {
                "plan_vector": [str(plan_vec)] * 6,
                "condition_vector": [str(cond_vec)] * 6,
            }
        )
        df.to_csv(data_path, index=False)

        monkeypatch.setattr(train_gan, "TRAINING_DATA_PATH", data_path)
        monkeypatch.setattr(
            train_gan, "FALLBACK_TRAINING_DATA_PATH", tmp_path / "fallback.csv"
        )

        _, train_c, _, val_c = train_gan.load_real_data()
        assert train_c.shape == (4, train_gan.CONDITION_DIM)  # 80% of 6 = 4
        assert val_c.shape == (2, train_gan.CONDITION_DIM)
        # Non-zero condition values should be preserved
        assert train_c[0, 0] == pytest.approx(0.0)
        assert train_c[0, 1] == pytest.approx(1.0)

    def test_missing_condition_vector_uses_zero_conditions(self, tmp_path, monkeypatch):
        import pandas as pd

        data_path = tmp_path / "training_data.csv"
        plan_vec = [0.5] * train_gan.PLAN_DIM
        df = pd.DataFrame({"plan_vector": [str(plan_vec)] * 6})
        df.to_csv(data_path, index=False)

        monkeypatch.setattr(train_gan, "TRAINING_DATA_PATH", data_path)
        monkeypatch.setattr(
            train_gan, "FALLBACK_TRAINING_DATA_PATH", tmp_path / "fallback.csv"
        )

        _, train_c, _, _ = train_gan.load_real_data()
        assert train_c.shape == (4, train_gan.CONDITION_DIM)
        assert (train_c == 0).all()

    def test_plan_vector_column_slice_fallback(self, tmp_path, monkeypatch):
        """When plan_vector column absent, first PLAN_DIM columns are used."""
        import pandas as pd

        data_path = tmp_path / "training_data.csv"
        cols = [f"f{i}" for i in range(train_gan.PLAN_DIM)]
        df = pd.DataFrame({c: [0.5] * 6 for c in cols})
        df.to_csv(data_path, index=False)

        monkeypatch.setattr(train_gan, "TRAINING_DATA_PATH", data_path)
        monkeypatch.setattr(
            train_gan, "FALLBACK_TRAINING_DATA_PATH", tmp_path / "fallback.csv"
        )

        train, _, _, _ = train_gan.load_real_data()
        assert train.shape == (4, train_gan.PLAN_DIM)


# ===========================================================================
# Module constants
# ===========================================================================


class TestModuleConstants:
    def test_plan_dim(self):
        assert train_gan.PLAN_DIM == 19

    def test_condition_dim(self):
        assert train_gan.CONDITION_DIM == 32

    def test_latent_dim(self):
        assert train_gan.LATENT_DIM == 64

    def test_script_dir_is_path(self):
        from pathlib import Path

        assert isinstance(train_gan.SCRIPT_DIR, Path)
        assert train_gan.SCRIPT_DIR.exists()
