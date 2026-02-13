# Release Hardening Checklist

- [ ] `make lint-fast test-fast sast-fast` is green
- [ ] contract docs are current with implementation
- [ ] `wrkr demo`, `wrkr submit`, `wrkr accept run`, `wrkr export`, `wrkr verify` smoke tests pass
- [ ] serve hardening behavior verified (loopback default, non-loopback guardrails)
- [ ] release notes include breaking/non-breaking contract changes
