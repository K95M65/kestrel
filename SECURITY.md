# Security

Found a vulnerability in **Kestrel**? Report it privately on
[K95M65/kestrel](https://github.com/K95M65/kestrel/security/advisories/new)
instead of filing a public issue.

Issues in **upstream Autonomous OS** still go to
[autonomous-ai/autonomous-os](https://github.com/autonomous-ai/autonomous-os/security/advisories/new).

One thing to get right in production: the on-device HAL authenticates with a
`device_auth_token` that is **separate from your LLM provider key** — keep them distinct so
leaking a model-billing key can't hand someone control of the hardware.
