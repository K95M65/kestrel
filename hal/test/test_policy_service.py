"""Tests for the policy interface before learned-policy execution exists."""
import logging
import unittest
from unittest.mock import Mock

from hal.policy.service import LoggingPolicyService


class TestLoggingPolicyService(unittest.TestCase):
    def setUp(self):
        self.logger = Mock(spec=logging.Logger)
        self.service = LoggingPolicyService(self.logger)

    def test_run_is_explicitly_dry_run_and_only_records_intent(self):
        run = self.service.run("lerobot/smolvla_base", "pick up the mug")

        self.assertTrue(run.dry_run)
        self.assertEqual(run.state, "dry_run")
        self.assertEqual(run.policy, "lerobot/smolvla_base")
        self.assertIs(self.service.active_run(), run)
        self.logger.info.assert_called_once()
        self.assertIn("no inference or motor command issued", self.logger.info.call_args.args[0])

    def test_only_one_dry_run_can_be_active(self):
        self.service.run("lerobot/act", "open the drawer")

        with self.assertRaisesRegex(RuntimeError, "already active"):
            self.service.run("lerobot/smolvla_base", "pick up the mug")

    def test_stop_forgets_the_dry_run_without_actuating(self):
        run = self.service.run("lerobot/act", "open the drawer")

        self.assertIs(self.service.stop(), run)
        self.assertIsNone(self.service.active_run())
        self.assertIn("no motor command issued", self.logger.info.call_args.args[0])

