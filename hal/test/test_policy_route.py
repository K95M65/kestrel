"""HTTP contract tests for the policy dry-run route (no hardware required)."""
import logging
import unittest

from fastapi import FastAPI
from fastapi.testclient import TestClient

import hal.app_state as state
from hal.policy.service import LoggingPolicyService
from hal.routes.policy import router


class TestPolicyRoute(unittest.TestCase):
    def setUp(self):
        self.previous_service = state.policy_service
        state.policy_service = LoggingPolicyService(logging.getLogger("hal.test.policy"))
        app = FastAPI()
        app.include_router(router)
        self.client = TestClient(app)

    def tearDown(self):
        state.policy_service = self.previous_service

    def test_accepts_and_reports_a_dry_run_without_hardware(self):
        response = self.client.post(
            "/policy/run",
            json={"policy": "lerobot/smolvla_base", "task": "pick up the mug"},
        )

        self.assertEqual(response.status_code, 200)
        body = response.json()
        self.assertEqual(body["status"], "accepted")
        self.assertEqual(body["state"], "dry_run")
        self.assertTrue(body["dry_run"])
        self.assertEqual(body["policy"], "lerobot/smolvla_base")

        status = self.client.get("/policy")
        self.assertEqual(status.status_code, 200)
        self.assertEqual(status.json()["active"]["id"], body["id"])

    def test_rejects_a_second_active_dry_run(self):
        first = {"policy": "lerobot/act", "task": "open the drawer"}
        self.assertEqual(self.client.post("/policy/run", json=first).status_code, 200)

        second = {"policy": "lerobot/smolvla_base", "task": "pick up the mug"}
        self.assertEqual(self.client.post("/policy/run", json=second).status_code, 409)

    def test_validates_required_request_fields(self):
        response = self.client.post("/policy/run", json={"policy": "", "task": ""})
        self.assertEqual(response.status_code, 422)
